package adapter

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type commandSourceLocker struct {
	mu   sync.Mutex
	held map[string]string
}

func newCommandSourceLocker() *commandSourceLocker {
	return &commandSourceLocker{held: make(map[string]string)}
}

func (l *commandSourceLocker) Lock(_ context.Context, key, owner string, _ time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.held[key]; exists {
		return false, nil
	}
	l.held[key] = owner
	return true, nil
}

func (l *commandSourceLocker) Unlock(_ context.Context, key, owner string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held[key] == owner {
		delete(l.held, key)
	}
	return nil
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
	if err := db.AutoMigrate(new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate command source models: %v", err)
	}
	return db
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
