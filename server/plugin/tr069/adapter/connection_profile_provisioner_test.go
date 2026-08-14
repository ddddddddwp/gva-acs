package adapter

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type recordingCredentialScheduler struct {
	deviceIDs []uint
	accept    bool
}

func (s *recordingCredentialScheduler) Schedule(deviceID uint) bool {
	s.deviceIDs = append(s.deviceIDs, deviceID)
	return s.accept
}

func newProvisionerTest(t *testing.T, wakeup service.CommandWakeupFunc) (*ConnectionCredentialProvisioner, *ConnectionProfileRepository, *gorm.DB) {
	t.Helper()
	previousRuntime := config.CurrentRuntime()
	settings := previousRuntime.Settings
	settings.ConnectionRequest = testCredentialConfig(0x71)
	settings.ConnectionRequest.AutoProvisionCredentials = true
	config.StoreRuntime(settings)
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate provisioner schema: %v", err)
	}
	repository := NewConnectionProfileRepository(db, NewRuntimeCredentialCipher())
	manager := service.NewCommandManager(db, wakeup,
		service.WithCommandPayloadProtector(NewConnectionProfilePayloadProtector(repository)),
	)
	return NewConnectionCredentialProvisioner(manager, repository), repository, db
}

func createProvisioningCandidate(t *testing.T, repository *ConnectionProfileRepository, db *gorm.DB, serial string) model.Device {
	t.Helper()
	device := model.Device{
		OUI:          "001122",
		SerialNumber: serial,
		LastInform:   time.Now(),
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	result, err := repository.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName: "http://127.0.0.1:8400",
	})
	if err != nil {
		t.Fatalf("collect candidate: %v", err)
	}
	if !result.NeedsProvisioning {
		t.Fatalf("candidate did not need provisioning: %#v", result)
	}
	return device
}

func loadProvisioningCommand(t *testing.T, db *gorm.DB, commandID string) model.Command {
	t.Helper()
	var command model.Command
	if err := db.First(&command, "command_id = ?", commandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	return command
}

func advanceProvisioningCommandToSent(t *testing.T, db *gorm.DB, commandID string) {
	t.Helper()
	command := loadProvisioningCommand(t, db, commandID)
	building, err := service.NewCommandStore(db).Transition(context.Background(), service.CommandTransition{
		CommandID:       commandID,
		FromStatuses:    []string{model.CommandStatusWaitingDevice},
		ToStatus:        model.CommandStatusBuilding,
		ExpectedVersion: command.Version,
		EventType:       "REQUEST_BUILDING",
	})
	if err != nil {
		t.Fatalf("advance command to BUILDING: %v", err)
	}
	if err := newGormCommandRepo(db).MarkSending(context.Background(), commandID, "cwmp-provision", time.Now()); err != nil {
		t.Fatalf("advance command version %d to SENT: %v", building.Version, err)
	}
}

func TestProvisionerCreatesOneSystemSPVProtectsPasswordAndMarksReady(t *testing.T) {
	provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
	device := createProvisioningCandidate(t, repository, db, "PROVISION-READY")

	first, err := provisioner.Ensure(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	second, err := provisioner.Ensure(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if first.CommandID == "" || second.CommandID != first.CommandID {
		t.Fatalf("results = %#v %#v", first, second)
	}
	var commandCount int64
	if err := db.Model(new(model.Command)).Where("device_id = ?", device.ID).Count(&commandCount).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	if commandCount != 1 {
		t.Fatalf("command count = %d, want 1", commandCount)
	}
	command := loadProvisioningCommand(t, db, first.CommandID)
	if command.Origin != model.CommandOriginSystem || command.Operation != "SetParameterValues" || command.DedupKey != "connection-profile:"+strconv.FormatUint(uint64(device.ID), 10)+":v1" {
		t.Fatalf("command = %#v", command)
	}
	username, password, err := repository.LoadCredential(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("load generated credential: %v", err)
	}
	if username == "" || password == "" {
		t.Fatalf("generated credential is empty: %q/%q", username, password)
	}
	if strings.Contains(string(command.ParamsJSON), password) || !strings.Contains(string(command.ParamsJSON), connectionRequestPasswordPlaceholder) {
		t.Fatalf("command payload was not protected: %s", command.ParamsJSON)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.ProvisionState != model.ConnectionProfileStateProvisioning || profile.ProvisionCommandID != first.CommandID || profile.CredentialSource != model.ConnectionCredentialSourceAuto {
		t.Fatalf("profile after SubmitSystem protector and hook = %#v", profile)
	}

	advanceProvisioningCommandToSent(t, db, first.CommandID)
	if err := newGormCommandRepo(db).MarkSuccess(context.Background(), first.CommandID, time.Now()); err != nil {
		t.Fatalf("MarkSuccess: %v", err)
	}
	if got := loadConnectionProfile(t, db, device.ID); got.ProvisionState != model.ConnectionProfileStateReady || got.ProvisionCommandID != first.CommandID || got.LastError != "" {
		t.Fatalf("ready profile = %#v", got)
	}
}

func TestProvisionerManualCredentialProtectionRemainsReady(t *testing.T) {
	_, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
	device := createProvisioningCandidate(t, repository, db, "PROVISION-MANUAL")
	request := req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{
		{Name: connectionRequestUsernameName, Type: "xsd:string", Value: "manual-user"},
		{Name: connectionRequestPasswordName, Type: "xsd:string", Value: "manual-secret"},
	}}
	encoded, err := service.EncodeRPCRequest("SetParameterValues", request)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	protector := NewConnectionProfilePayloadProtector(repository)
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, protectErr := protector.Protect(context.Background(), tx, device.ID, "SetParameterValues", model.CommandOriginUser, encoded)
		return protectErr
	}); err != nil {
		t.Fatalf("protect manual request: %v", err)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.ProvisionState != model.ConnectionProfileStateReady || profile.CredentialSource != model.ConnectionCredentialSourceManual || profile.ProvisionCommandID != "" {
		t.Fatalf("manual profile = %#v", profile)
	}
}

func TestProvisionerMarksProfileFailedOnCommandFailure(t *testing.T) {
	provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
	device := createProvisioningCandidate(t, repository, db, "PROVISION-FAIL")
	result, err := provisioner.Ensure(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if err := newGormCommandRepo(db).MarkFail(context.Background(), result.CommandID, 9003, "invalid credentials", time.Now()); err != nil {
		t.Fatalf("MarkFail: %v", err)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.ProvisionState != model.ConnectionProfileStateFailed || profile.LastError != "invalid credentials" || profile.ProvisionCommandID != result.CommandID {
		t.Fatalf("failed profile = %#v", profile)
	}
}

func TestProvisionerWakeupFailureDoesNotLeaveProfileProvisioning(t *testing.T) {
	wakeupErr := errors.New("redis unavailable")
	provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return wakeupErr })
	device := createProvisioningCandidate(t, repository, db, "PROVISION-WAKE-FAIL")
	result, err := provisioner.Ensure(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if result.CommandID == "" || result.Status != model.CommandStatusFailed {
		t.Fatalf("result = %#v", result)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.ProvisionState != model.ConnectionProfileStateFailed || profile.ProvisionCommandID != result.CommandID || !strings.Contains(profile.LastError, wakeupErr.Error()) {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestProvisionerDisabledAndMissingKeyCreateNoCommand(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
		device := createProvisioningCandidate(t, repository, db, "PROVISION-DISABLED")
		settings := config.CurrentRuntime().Settings
		settings.ConnectionRequest.AutoProvisionCredentials = false
		config.StoreRuntime(settings)
		result, err := provisioner.Ensure(context.Background(), device.ID)
		if err != nil || result.CommandID != "" {
			t.Fatalf("Ensure disabled = %#v, %v", result, err)
		}
		assertProvisionerCommandCount(t, db, 0)
		if got := loadConnectionProfile(t, db, device.ID).ProvisionState; got != model.ConnectionProfileStateDiscovered {
			t.Fatalf("disabled state = %s", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
		device := createProvisioningCandidate(t, repository, db, "PROVISION-MISSING-KEY")
		settings := config.CurrentRuntime().Settings
		settings.ConnectionRequest.CredentialEncryptionKey = ""
		config.StoreRuntime(settings)
		if _, err := provisioner.Ensure(context.Background(), device.ID); err == nil {
			t.Fatal("Ensure succeeded without credential key")
		}
		assertProvisionerCommandCount(t, db, 0)
		if got := loadConnectionProfile(t, db, device.ID).ProvisionState; got != model.ConnectionProfileStateDiscovered {
			t.Fatalf("missing-key state = %s", got)
		}
	})
}

func assertProvisionerCommandCount(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var got int64
	if err := db.Model(new(model.Command)).Count(&got).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	if got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
}

func TestProvisionerScheduleIsNonBlockingAndBounded(t *testing.T) {
	provisioner, _, _ := newProvisionerTest(t, func(context.Context, string) error { return nil })
	provisioner.queue = make(chan uint, 1)
	if !provisioner.Schedule(1) {
		t.Fatal("first Schedule was rejected")
	}
	done := make(chan bool, 1)
	go func() { done <- provisioner.Schedule(2) }()
	select {
	case accepted := <-done:
		if accepted {
			t.Fatal("full queue accepted another device")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Schedule blocked on a full queue")
	}
}

func TestProvisionerRunRecoversDiscoveredButDoesNotRetryFailed(t *testing.T) {
	provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
	discovered := createProvisioningCandidate(t, repository, db, "PROVISION-RECOVER")
	failed := createProvisioningCandidate(t, repository, db, "PROVISION-NO-RETRY")
	if err := db.Model(new(model.ConnectionProfile)).Where("device_id = ?", failed.ID).Updates(map[string]any{
		"provision_state": model.ConnectionProfileStateFailed,
		"last_error":      "previous failure",
	}).Error; err != nil {
		t.Fatalf("mark failed profile: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		provisioner.Run(ctx)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if loadConnectionProfile(t, db, discovered.ID).ProvisionState == model.ConnectionProfileStateProvisioning {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
	if got := loadConnectionProfile(t, db, discovered.ID).ProvisionState; got != model.ConnectionProfileStateProvisioning {
		t.Fatalf("recovered state = %s", got)
	}
	failedProfile := loadConnectionProfile(t, db, failed.ID)
	if failedProfile.ProvisionState != model.ConnectionProfileStateFailed || failedProfile.ProvisionCommandID != "" || failedProfile.LastError != "previous failure" {
		t.Fatalf("failed profile was retried: %#v", failedProfile)
	}
	var failedCommands int64
	if err := db.Model(new(model.Command)).Where("device_id = ?", failed.ID).Count(&failedCommands).Error; err != nil {
		t.Fatalf("count failed-device commands: %v", err)
	}
	if failedCommands != 0 {
		t.Fatalf("failed device commands = %d, want 0", failedCommands)
	}
}

func TestProvisionerRunReconcilesProvisioningProfileWhoseCommandIsTerminal(t *testing.T) {
	provisioner, repository, db := newProvisionerTest(t, func(context.Context, string) error { return nil })
	device := createProvisioningCandidate(t, repository, db, "PROVISION-TERMINAL-RECOVERY")
	result, err := provisioner.Ensure(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("Ensure(): %v", err)
	}
	command := loadProvisioningCommand(t, db, result.CommandID)
	finishedAt := time.Now()
	if _, err := service.NewCommandStore(db).Transition(context.Background(), service.CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{command.Status},
		ToStatus:        model.CommandStatusFailed,
		ExpectedVersion: command.Version,
		EventType:       "TEST_CRASH_WINDOW",
		Stage:           "redis.enqueue",
		Message:         "simulated terminal command before profile compensation",
		Updates: map[string]any{
			"failure_stage":     "redis.enqueue",
			"fault_string":      "simulated terminal command before profile compensation",
			"finished_at":       finishedAt,
			"phase_deadline_at": nil,
		},
	}); err != nil {
		t.Fatalf("terminate command without profile update: %v", err)
	}
	if got := loadConnectionProfile(t, db, device.ID).ProvisionState; got != model.ConnectionProfileStateProvisioning {
		t.Fatalf("precondition profile state = %s", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		provisioner.Run(ctx)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if loadConnectionProfile(t, db, device.ID).ProvisionState == model.ConnectionProfileStateFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after reconciliation")
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.ProvisionState != model.ConnectionProfileStateFailed || profile.ProvisionCommandID != command.CommandID || !strings.Contains(profile.LastError, "simulated terminal") {
		t.Fatalf("reconciled profile = %#v", profile)
	}
}
