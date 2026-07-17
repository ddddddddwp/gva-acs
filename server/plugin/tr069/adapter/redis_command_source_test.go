package adapter

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type commandSourceLocker struct {
	mu     sync.Mutex
	held   map[string]string
	owners []string
	ttls   []time.Duration
}

func newCommandSourceLocker() *commandSourceLocker {
	return &commandSourceLocker{held: make(map[string]string)}
}

func (l *commandSourceLocker) Lock(_ context.Context, key, owner string, ttl time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.held[key]; exists {
		return false, nil
	}
	l.held[key] = owner
	l.owners = append(l.owners, owner)
	l.ttls = append(l.ttls, ttl)
	return true, nil
}

func (l *commandSourceLocker) Unlock(_ context.Context, key, owner string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held[key] == owner {
		delete(l.held, key)
		return nil
	}
	return fmt.Errorf("%w: %s", ErrRedisLockOwnershipLost, key)
}

func (l *commandSourceLocker) expire(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.held, key)
}

func (l *commandSourceLocker) currentOwner(key string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.held[key]
}

func (l *commandSourceLocker) isHeld(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.held[key]
	return ok
}

func newRedisCommandSourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate command source models: %v", err)
	}
	return db
}

func TestRedisCommandSourceHydratesProtectedConnectionRequestPasswordOnlyInMemory(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	cipher, err := NewCredentialCipher(config.ConnectionRequestConfig{
		CredentialKeyVersion:    "v1",
		CredentialEncryptionKey: base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
	})
	if err != nil {
		t.Fatalf("create credential cipher: %v", err)
	}
	repository := NewConnectionProfileRepository(db, cipher)
	if _, err := repository.UpdateOverride(context.Background(), 31, ConnectionProfileOverride{
		OverrideURL: "http://127.0.0.1:8400/",
		Username:    "acs-user",
		Password:    "runtime-only-secret",
	}); err != nil {
		t.Fatalf("seed connection profile: %v", err)
	}

	now := time.Now().Add(-time.Second)
	params, err := service.EncodeRPCRequest("SetParameterValues", req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{
		{Name: connectionRequestUsernameName, Value: "acs-user", Type: "xsd:string"},
		{Name: connectionRequestPasswordName, Value: connectionRequestPasswordPlaceholder, Type: "xsd:string"},
	}})
	if err != nil {
		t.Fatalf("encode protected request: %v", err)
	}
	command := model.Command{
		CommandID: "hydrate-password", DeviceID: 31, DeviceKey: "001122-HYDRATE", Operation: "SetParameterValues",
		ParamsJSON: params, Status: model.CommandStatusWaitingDevice,
		QueuedAt: now, WaitingAt: &now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed protected command: %v", err)
	}

	source := newRedisCommandSource(db, newCommandSourceLocker(), RedisCommandSourceConfig{InstanceID: "hydrate-test"},
		WithRedisCommandHydrator(NewConnectionProfilePayloadProtector(repository)))
	pulled, ack, _, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil {
		t.Fatalf("Pull(): %v", err)
	}
	if pulled == nil || ack == nil {
		t.Fatalf("Pull() = command:%#v ack:%v", pulled, ack != nil)
	}
	parameters, ok := pulled.Params["parameters"].([]map[string]interface{})
	if !ok {
		t.Fatalf("pulled parameters type = %T", pulled.Params["parameters"])
	}
	var password string
	for _, parameter := range parameters {
		if parameter["name"] == connectionRequestPasswordName {
			password, _ = parameter["value"].(string)
		}
	}
	if password != "runtime-only-secret" {
		t.Fatalf("hydrated password = %q", password)
	}
	var persisted model.Command
	if err := db.First(&persisted, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("reload protected command: %v", err)
	}
	if strings.Contains(string(persisted.ParamsJSON), "runtime-only-secret") {
		t.Fatal("plaintext password was persisted after hydration")
	}
	if err := ack(context.Background()); err != nil {
		t.Fatalf("ack hydrated command: %v", err)
	}
}

func TestRedisCommandSourceUsesUniqueOwnerPerAcquisitionAndConfiguredTTL(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	locker := newCommandSourceLocker()
	ttl := 47 * time.Second
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "source-instance", LockTTL: ttl})

	for range 2 {
		cmd, ack, nack, err := source.Pull(context.Background(), "001122-NO-COMMAND")
		if err != nil || cmd != nil || ack != nil || nack != nil {
			t.Fatalf("empty Pull() = command:%#v ack:%v nack:%v error:%v", cmd, ack != nil, nack != nil, err)
		}
	}
	if len(locker.owners) != 2 || locker.owners[0] == locker.owners[1] {
		t.Fatalf("lock owners = %#v, want two unique acquisition tokens", locker.owners)
	}
	for _, owner := range locker.owners {
		if !strings.HasPrefix(owner, "source-instance:") {
			t.Fatalf("owner = %q, want source-instance UUID prefix", owner)
		}
	}
	if !reflect.DeepEqual(locker.ttls, []time.Duration{ttl, ttl}) {
		t.Fatalf("lock TTLs = %#v, want both %s", locker.ttls, ttl)
	}
}

func TestValidateRedisLockReleaseReportsLostOwnershipOnZeroDelete(t *testing.T) {
	if err := validateRedisLockRelease("tr069:lock:device:test", 0); !errors.Is(err, ErrRedisLockOwnershipLost) {
		t.Fatalf("zero-delete error = %v, want ownership-lost", err)
	}
	if err := validateRedisLockRelease("tr069:lock:device:test", 1); err != nil {
		t.Fatalf("single-delete error = %v, want nil", err)
	}
}

func TestRedisCommandSourceStaleAckCannotReleaseNewAcquisition(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Now().Add(-time.Minute)
	command := model.Command{
		CommandID: "stale-ack", DeviceID: 21, DeviceKey: "001122-STALE", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingDevice,
		QueuedAt: now, WaitingAt: &now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	locker := newCommandSourceLocker()
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "stale-test", LockTTL: time.Second})
	_, staleAck, _, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil || staleAck == nil {
		t.Fatalf("first Pull() ack/error = %v/%v", staleAck != nil, err)
	}
	lockKey := RedisDeviceLockPrefix + command.DeviceKey
	firstOwner := locker.currentOwner(lockKey)
	locker.expire(lockKey)

	var building model.Command
	if err := db.First(&building, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load first BUILDING state: %v", err)
	}
	waitingAt := time.Now()
	deadline := waitingAt.Add(time.Minute)
	if _, err := service.NewCommandStore(db).Transition(context.Background(), service.CommandTransition{
		CommandID: building.CommandID, FromStatuses: []string{model.CommandStatusBuilding},
		ToStatus: model.CommandStatusWaitingDevice, ExpectedVersion: building.Version,
		EventType: "STALE_LOCK_RECOVERED", Stage: "test", Updates: map[string]any{
			"building_at": nil, "waiting_at": waitingAt, "phase_deadline_at": deadline,
		},
	}); err != nil {
		t.Fatalf("recover expired BUILDING state: %v", err)
	}
	_, currentAck, _, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil || currentAck == nil {
		t.Fatalf("second Pull() ack/error = %v/%v", currentAck != nil, err)
	}
	secondOwner := locker.currentOwner(lockKey)
	if firstOwner == secondOwner {
		t.Fatalf("owners reused across acquisitions: %q", firstOwner)
	}

	if err := staleAck(context.Background()); !errors.Is(err, ErrRedisLockOwnershipLost) {
		t.Fatalf("stale ack error = %v, want ownership-lost", err)
	}
	if got := locker.currentOwner(lockKey); got != secondOwner {
		t.Fatalf("stale ack changed current owner from %q to %q", secondOwner, got)
	}
	if err := currentAck(context.Background()); err != nil {
		t.Fatalf("current ack: %v", err)
	}
	if locker.isHeld(lockKey) {
		t.Fatal("current acquisition lock remained after matching ack")
	}
}

func TestCommandManagerWakeFailureCompensatesConcurrentPull(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "WAKE-PULL-RACE", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	wakeupEntered := make(chan struct{})
	releaseWakeup := make(chan struct{})
	wakeupErr := errors.New("redis enqueue failed after concurrent pull")
	manager := service.NewCommandManager(db, func(context.Context, string) error {
		close(wakeupEntered)
		<-releaseWakeup
		return wakeupErr
	}, service.WithCommandManagerNow(func() time.Time { return now }))
	type submitOutcome struct {
		result service.SubmitResult
		err    error
	}
	submitted := make(chan submitOutcome, 1)
	go func() {
		result, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
		submitted <- submitOutcome{result: result, err: err}
	}()
	<-wakeupEntered

	locker := newCommandSourceLocker()
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "wake-pull-race"})
	pulled, ack, _, err := source.Pull(context.Background(), "001122-WAKE-PULL-RACE")
	if err != nil || pulled == nil || ack == nil {
		t.Fatalf("concurrent Pull() = command:%#v ack:%v error:%v", pulled, ack != nil, err)
	}
	close(releaseWakeup)
	outcome := <-submitted
	if outcome.err != nil {
		t.Fatalf("Submit() error = %v, want compensated terminal result", outcome.err)
	}
	if outcome.result.Status != model.CommandStatusFailed {
		t.Fatalf("Submit() status = %q, want FAILED", outcome.result.Status)
	}
	if err := ack(context.Background()); err != nil {
		t.Fatalf("release concurrent Pull lock: %v", err)
	}
	var failed model.Command
	if err := db.First(&failed, "command_id = ?", outcome.result.CommandID).Error; err != nil {
		t.Fatalf("load compensated command: %v", err)
	}
	if failed.Status != model.CommandStatusFailed || failed.FailureStage != "redis.enqueue" {
		t.Fatalf("compensated concurrent state = %s/%q, want FAILED/redis.enqueue", failed.Status, failed.FailureStage)
	}
	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", failed.CommandID, "DISPATCH_FAILED").First(&event).Error; err != nil {
		t.Fatalf("load concurrent compensation event: %v", err)
	}
	if event.FromStatus != model.CommandStatusBuilding || event.ToStatus != model.CommandStatusFailed {
		t.Fatalf("concurrent compensation transition = %s -> %s, want BUILDING -> FAILED", event.FromStatus, event.ToStatus)
	}
}

func TestRedisCommandSourcePullUsesDatabaseHeadAndTypedParams(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC)
	request := req.GetParameterValuesRequest{Paths: []string{"Device.DeviceInfo.SerialNumber", "Device.DeviceInfo.SoftwareVersion"}}
	paramsJSON, err := service.EncodeRPCRequest("GetParameterValues", request)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	command := model.Command{
		CommandID:  "database-head",
		DeviceID:   1,
		DeviceKey:  "001122-SOURCE",
		Operation:  "GetParameterValues",
		ParamsJSON: model.LongTextJSON(paramsJSON),
		Status:     model.CommandStatusWaitingDevice,
		QueuedAt:   now.Add(-time.Minute),
		WaitingAt:  &now,
		CreatedAt:  now.Add(-time.Minute),
		UpdatedAt:  now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	locker := newCommandSourceLocker()
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "source-test", LockTTL: time.Minute})
	pulled, ack, nack, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil {
		t.Fatalf("Pull() error: %v", err)
	}
	if pulled == nil || ack == nil || nack == nil {
		t.Fatalf("Pull() callbacks = command:%#v ack:%v nack:%v", pulled, ack != nil, nack != nil)
	}
	if pulled.ID != command.CommandID || pulled.DeviceKey != command.DeviceKey || pulled.Operation != command.Operation {
		t.Fatalf("pulled command = %#v, want database row %#v", pulled, command)
	}
	if want := request.Paths; !reflect.DeepEqual(pulled.Params["paths"], want) {
		t.Fatalf("typed paths = %#v (%T), want %#v (%T)", pulled.Params["paths"], pulled.Params["paths"], want, want)
	}

	var building model.Command
	if err := db.First(&building, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load building command: %v", err)
	}
	if building.Status != model.CommandStatusBuilding || building.BuildingAt == nil {
		t.Fatalf("database state after Pull = %s buildingAt:%v", building.Status, building.BuildingAt)
	}
	lockKey := RedisDeviceLockPrefix + command.DeviceKey
	if !locker.isHeld(lockKey) {
		t.Fatal("device lock released before ack")
	}
	if err := ack(context.Background()); err != nil {
		t.Fatalf("ack: %v", err)
	}
	if locker.isHeld(lockKey) {
		t.Fatal("device lock still held after ack")
	}
}

func TestRedisCommandSourceFIFOBlocksQueuedSentAndWaitingTransferHeads(t *testing.T) {
	for _, status := range []string{model.CommandStatusQueued, model.CommandStatusSent, model.CommandStatusWaitingTransfer} {
		t.Run(status, func(t *testing.T) {
			db := newRedisCommandSourceTestDB(t)
			now := time.Now()
			head := model.Command{
				CommandID:  "blocking-head-" + status,
				DeviceID:   10,
				DeviceKey:  "001122-BLOCK-" + status,
				Operation:  "GetRPCMethods",
				ParamsJSON: model.LongTextJSON(`{}`),
				Status:     status,
				QueuedAt:   now.Add(-time.Minute),
				CreatedAt:  now.Add(-time.Minute),
			}
			if err := service.NewCommandStore(db).Create(context.Background(), &head); err != nil {
				t.Fatalf("seed blocking head: %v", err)
			}
			later := model.Command{
				CommandID:  "later-waiting-" + status,
				DeviceID:   head.DeviceID,
				DeviceKey:  head.DeviceKey,
				Operation:  "GetRPCMethods",
				ParamsJSON: model.LongTextJSON(`{}`),
				Status:     model.CommandStatusWaitingDevice,
				QueuedAt:   now,
				CreatedAt:  now,
			}
			if err := service.NewCommandStore(db).Create(context.Background(), &later); err != nil {
				t.Fatalf("seed later command: %v", err)
			}

			locker := newCommandSourceLocker()
			source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "blocking-test"})
			pulled, ack, nack, err := source.Pull(context.Background(), head.DeviceKey)
			if err != nil {
				t.Fatalf("Pull() error: %v", err)
			}
			if pulled != nil || ack != nil || nack != nil {
				t.Fatalf("Pull() bypassed %s head: command=%#v ack=%v nack=%v", status, pulled, ack != nil, nack != nil)
			}
			if locker.isHeld(RedisDeviceLockPrefix + head.DeviceKey) {
				t.Fatal("device lock retained after blocked Pull")
			}

			var gotLater model.Command
			if err := db.First(&gotLater, "command_id = ?", later.CommandID).Error; err != nil {
				t.Fatalf("load later command: %v", err)
			}
			if gotLater.Status != model.CommandStatusWaitingDevice {
				t.Fatalf("later status = %s, want unchanged WAITING_DEVICE", gotLater.Status)
			}
		})
	}
}

func TestRedisCommandSourceNackRestoresBuildingToWaitingDevice(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Now().Add(-time.Minute)
	command := model.Command{
		CommandID:  "nack-restores",
		DeviceID:   11,
		DeviceKey:  "001122-NACK",
		Operation:  "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusWaitingDevice,
		QueuedAt:   now,
		WaitingAt:  &now,
		CreatedAt:  now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	locker := newCommandSourceLocker()
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "nack-test"})
	pulled, _, nack, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil || pulled == nil || nack == nil {
		t.Fatalf("Pull() = command:%#v nack:%v error:%v", pulled, nack != nil, err)
	}
	if err := nack(context.Background(), "request was not sent"); err != nil {
		t.Fatalf("nack: %v", err)
	}

	var restored model.Command
	if err := db.First(&restored, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load restored command: %v", err)
	}
	if restored.Status != model.CommandStatusWaitingDevice || restored.BuildingAt != nil || restored.WaitingAt == nil || restored.PhaseDeadlineAt == nil {
		t.Fatalf("restored state = %s building:%v waiting:%v deadline:%v", restored.Status, restored.BuildingAt, restored.WaitingAt, restored.PhaseDeadlineAt)
	}
	if locker.isHeld(RedisDeviceLockPrefix + command.DeviceKey) {
		t.Fatal("device lock retained after nack")
	}
}

func TestRedisCommandSourceNackDoesNotRestoreAfterRequestWasSent(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Now().Add(-time.Minute)
	command := model.Command{
		CommandID: "nack-after-send", DeviceID: 13, DeviceKey: "001122-SENT-NACK", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingDevice,
		QueuedAt: now, WaitingAt: &now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	locker := newCommandSourceLocker()
	source := newRedisCommandSource(db, locker, RedisCommandSourceConfig{InstanceID: "sent-nack-test"})
	pulled, _, nack, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil || pulled == nil || nack == nil {
		t.Fatalf("Pull() = command:%#v nack:%v error:%v", pulled, nack != nil, err)
	}
	if err := newGormCommandRepo(db).MarkSending(context.Background(), command.CommandID, "cwmp-sent", time.Now()); err != nil {
		t.Fatalf("MarkSending(): %v", err)
	}
	if err := nack(context.Background(), "late cleanup"); err != nil {
		t.Fatalf("nack after send: %v", err)
	}
	var sent model.Command
	if err := db.First(&sent, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load sent command: %v", err)
	}
	if sent.Status != model.CommandStatusSent || sent.RequestID != "cwmp-sent" {
		t.Fatalf("nack restored already sent command: %#v", sent)
	}
	if locker.isHeld(RedisDeviceLockPrefix + command.DeviceKey) {
		t.Fatal("device lock retained after late nack")
	}
}

func TestRedisCommandSourceInjectsTransferCommandKeyAtDispatch(t *testing.T) {
	db := newRedisCommandSourceTestDB(t)
	now := time.Now()
	request := req.DownloadRequest{FileType: "1 Firmware Upgrade Image", URL: "https://example.com/fw.bin", FileSize: 4096}
	paramsJSON, err := service.EncodeRPCRequest("Download", request)
	if err != nil {
		t.Fatalf("encode Download: %v", err)
	}
	commandKey := "rpc-system-generated"
	command := model.Command{
		CommandID:  "download-with-key",
		DeviceID:   12,
		DeviceKey:  "001122-DOWNLOAD",
		Operation:  "Download",
		ParamsJSON: model.LongTextJSON(paramsJSON),
		CommandKey: &commandKey,
		Status:     model.CommandStatusWaitingDevice,
		QueuedAt:   now,
		WaitingAt:  &now,
		CreatedAt:  now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed Download: %v", err)
	}
	source := newRedisCommandSource(db, newCommandSourceLocker(), RedisCommandSourceConfig{InstanceID: "transfer-test"})
	pulled, ack, _, err := source.Pull(context.Background(), command.DeviceKey)
	if err != nil {
		t.Fatalf("Pull() error: %v", err)
	}
	if pulled == nil || pulled.Params["commandKey"] != commandKey {
		t.Fatalf("dispatched CommandKey = %#v, want %q", pulled, commandKey)
	}
	if !reflect.DeepEqual(pulled.Params["fileSize"], request.FileSize) {
		t.Fatalf("typed fileSize = %#v (%T), want %d", pulled.Params["fileSize"], pulled.Params["fileSize"], request.FileSize)
	}
	if err := ack(context.Background()); err != nil {
		t.Fatalf("ack: %v", err)
	}
}

func TestRedisCommandSourceInjectsServerCommandKeyAtDispatch(t *testing.T) {
	tests := []struct {
		operation string
		request   any
	}{
		{"Download", req.DownloadRequest{FileType: "1 Firmware Upgrade Image", URL: "https://example.test/fw.bin"}},
		{"Reboot", nil},
	}
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			db := newRedisCommandSourceTestDB(t)
			paramsJSON, err := service.EncodeRPCRequest(tt.operation, tt.request)
			if err != nil {
				t.Fatalf("encode %s: %v", tt.operation, err)
			}
			now := time.Now()
			key := "rpc-server-owned-" + strings.ToLower(tt.operation)
			command := model.Command{
				CommandID: "dispatch-" + tt.operation, DeviceID: 1,
				DeviceKey: "001122-" + strings.ToUpper(tt.operation),
				Operation: tt.operation, ParamsJSON: paramsJSON, CommandKey: &key,
				Status: model.CommandStatusWaitingDevice, QueuedAt: now,
				WaitingAt: &now, CreatedAt: now,
			}
			if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
				t.Fatalf("seed command: %v", err)
			}
			source := newRedisCommandSource(db, newCommandSourceLocker(),
				RedisCommandSourceConfig{InstanceID: "server-key-test"})
			pulled, ack, _, err := source.Pull(context.Background(), command.DeviceKey)
			if err != nil {
				t.Fatalf("Pull: %v", err)
			}
			if pulled == nil || pulled.Params["commandKey"] != key {
				t.Fatalf("dispatched params = %#v, want commandKey %q", pulled, key)
			}
			if err := ack(context.Background()); err != nil {
				t.Fatalf("ack: %v", err)
			}
		})
	}
}
