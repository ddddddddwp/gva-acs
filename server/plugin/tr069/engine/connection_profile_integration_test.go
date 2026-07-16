package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/observability"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/ddddddddwp/tr069-core-only/pkg/core/defaults"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type recordingEngineEventSink struct {
	mu     sync.Mutex
	events []observability.Event
}

func (*recordingEngineEventSink) Enabled(level observability.Level) bool {
	return level == observability.LevelInfo
}

func (s *recordingEngineEventSink) Emit(_ context.Context, event observability.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *recordingEngineEventSink) directions() map[observability.Direction]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[observability.Direction]bool)
	for _, event := range s.events {
		if event.Stage == "wire.xml" && len(event.Payload) > 0 {
			result[event.Direction] = true
		}
	}
	return result
}

func TestEngineDefaultParserAndBuilderShareEventSink(t *testing.T) {
	sink := new(recordingEngineEventSink)
	inflight := adapter.NewMemoryInflightRepo(time.Minute)
	commandRepo := defaults.NewMemoryCommandRepo()
	engine, err := New(Deps{
		EventSink:    sink,
		SessionStore: defaults.NewMemorySessionStore(0),
		DeviceRepo:   defaults.NewMemoryDeviceRepo(),
		CommandRepo:  commandRepo,
		CommandQueue: defaults.NewMemoryQueue(),
		InflightRepo: inflight,
		Hook:         defaults.NewInflightCorrelationHook(inflight, commandRepo),
	})
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	requestContext := adapter.WithDeviceMeta(context.Background(), &tr069.DeviceID{
		Manufacturer: "Manufacturer", OUI: "001122", ProductClass: "ProductClass", SerialNumber: "1234567890",
	}, "127.0.0.1")
	if _, err := engine.Handle(requestContext, &core.Request{
		ID: "sink-inform", RemoteIP: "127.0.0.1", Body: []byte(sampleInformXMLBoot2), Headers: map[string]string{},
	}); err != nil {
		t.Fatalf("Handle Inform: %v", err)
	}
	directions := sink.directions()
	if !directions[observability.DirectionInbound] || !directions[observability.DirectionOutbound] {
		t.Fatalf("wire event directions = %#v, want inbound and outbound", directions)
	}
}

func TestEngineDefaultConnectionProfileRuntimeProvisionsEncryptedSystemCommand(t *testing.T) {
	previousDB := global.GVA_DB
	previousRuntime := config.CurrentRuntime()
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		config.StoreRuntime(previousRuntime.Settings)
	})

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile),
		new(model.Command), new(model.CommandEvent), new(model.CommandXML),
	); err != nil {
		t.Fatalf("migrate integration schema: %v", err)
	}
	global.GVA_DB = db
	settings := previousRuntime.Settings
	settings.ConnectionRequest.AutoProvisionCredentials = true
	settings.ConnectionRequest.CredentialKeyVersion = "integration-v1"
	settings.ConnectionRequest.CredentialEncryptionKey = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x6b}, 32))
	config.StoreRuntime(settings)

	runCtx, cancel := context.WithCancel(context.Background())
	inflight := adapter.NewMemoryInflightRepo(time.Minute)
	engine, runtimeDone, err := newEngine(Deps{
		RuntimeContext: runCtx,
		CommandWakeup:  func(context.Context, string) error { return nil },
		SessionStore:   defaults.NewMemorySessionStore(0),
		CommandQueue:   defaults.NewMemoryQueue(),
		InflightRepo:   inflight,
	})
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if runtimeDone == nil {
		t.Fatal("default profile runtime did not expose lifecycle completion")
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-runtimeDone:
		case <-time.After(time.Second):
			t.Error("profile runtime did not stop after context cancellation")
		}
	})
	inform := strings.Replace(sampleInformXMLBoot2,
		`<ParameterList soap_enc:arrayType="cwmp:ParameterValueStruct[1]">`,
		`<ParameterList soap_enc:arrayType="cwmp:ParameterValueStruct[2]">`, 1)
	inform = strings.Replace(inform, `</ParameterList>`, `<ParameterValueStruct>
          <Name>Device.ManagementServer.ConnectionRequestURL</Name>
          <Value xsi:type="xsd:string">http://127.0.0.1:8400</Value>
        </ParameterValueStruct></ParameterList>`, 1)
	requestContext := adapter.WithDeviceMeta(context.Background(), &tr069.DeviceID{
		Manufacturer: "Manufacturer", OUI: "001122", ProductClass: "ProductClass", SerialNumber: "1234567890",
	}, "127.0.0.1")
	if _, err := engine.Handle(requestContext, &core.Request{
		ID: "profile-inform", RemoteIP: "127.0.0.1", Body: []byte(inform), Headers: map[string]string{},
	}); err != nil {
		t.Fatalf("Handle Inform: %v", err)
	}

	var command model.Command
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err = db.Where("origin = ? AND operation = ?", model.CommandOriginSystem, "SetParameterValues").First(&command).Error
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		var deviceCount, profileCount int64
		_ = db.Model(new(model.Device)).Count(&deviceCount).Error
		_ = db.Model(new(model.ConnectionProfile)).Count(&profileCount).Error
		var observed model.ConnectionProfile
		_ = db.First(&observed).Error
		t.Fatalf("load automatic system command: %v (devices=%d profiles=%d profileState=%q url=%t)", err, deviceCount, profileCount, observed.ProvisionState, observed.DiscoveredURL != "")
	}
	if strings.Contains(string(command.ParamsJSON), "generated-secret") || !strings.Contains(string(command.ParamsJSON), "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__") {
		t.Fatalf("persisted system params are not protected: %s", command.ParamsJSON)
	}
	var profile model.ConnectionProfile
	if err := db.First(&profile, "device_id = ?", command.DeviceID).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.ProvisionState != model.ConnectionProfileStateProvisioning || len(profile.PasswordCiphertext) == 0 {
		t.Fatalf("profile before terminal response = %#v", profile)
	}
	cipher := adapter.NewRuntimeCredentialCipher()
	password, err := cipher.Decrypt(profile.CredentialKeyVersion, profile.PasswordCiphertext)
	if err != nil || password == "" || strings.Contains(string(command.ParamsJSON), password) {
		t.Fatalf("encrypted credential isolation failed: passwordLength=%d err=%v", len(password), err)
	}

	store := service.NewCommandStore(db)
	building, err := store.Transition(context.Background(), service.CommandTransition{
		CommandID: command.CommandID, FromStatuses: []string{command.Status}, ToStatus: model.CommandStatusBuilding,
		ExpectedVersion: command.Version, EventType: "TEST_BUILDING", Stage: "core.build",
	})
	if err != nil {
		t.Fatalf("transition BUILDING: %v", err)
	}
	sentAt := time.Now()
	if err := new(adapter.GormCommandRepo).MarkSending(context.Background(), building.CommandID, "cwmp-integration", sentAt); err != nil {
		t.Fatalf("MarkSending: %v", err)
	}
	if err := new(adapter.GormCommandRepo).MarkSuccess(context.Background(), building.CommandID, time.Now()); err != nil {
		t.Fatalf("MarkSuccess: %v", err)
	}
	if err := db.First(&profile, "device_id = ?", command.DeviceID).Error; err != nil {
		t.Fatalf("reload terminal profile: %v", err)
	}
	if profile.ProvisionState != model.ConnectionProfileStateReady {
		t.Fatalf("terminal profile state = %q, want READY", profile.ProvisionState)
	}
}
