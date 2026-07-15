package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newCommandManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate command manager models: %v", err)
	}
	return db
}

func createCommandManagerDevice(t *testing.T, db *gorm.DB, serial string, now time.Time) model.Device {
	t.Helper()

	device := model.Device{OUI: "001122", SerialNumber: serial, LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	return device
}

func setCommandManagerCapabilities(t *testing.T, db *gorm.DB, deviceID uint, methods string) {
	t.Helper()
	if err := db.Create(&model.DeviceRPCMethods{DeviceID: deviceID, MethodsJSON: datatypes.JSON(methods)}).Error; err != nil {
		t.Fatalf("create device capabilities: %v", err)
	}
}

func TestCommandManagerCommandAndInitialEventAreOneTransaction(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "ATOMIC", now)
	injected := errors.New("injected initial event failure")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_manager_initial_event", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "CommandEvent" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register event failure: %v", err)
	}

	wakeups := 0
	manager := NewCommandManager(db, func(context.Context, string) error {
		wakeups++
		return nil
	}, WithCommandManagerNow(func() time.Time { return now }))
	if _, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil); !errors.Is(err, injected) {
		t.Fatalf("Submit() error = %v, want %v", err, injected)
	}

	var commands, events int64
	if err := db.Model(new(model.Command)).Count(&commands).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	if err := db.Model(new(model.CommandEvent)).Count(&events).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if commands != 0 || events != 0 {
		t.Fatalf("rows after rollback: commands=%d events=%d", commands, events)
	}
	if wakeups != 0 {
		t.Fatalf("wakeups = %d, want 0 before transaction commits", wakeups)
	}
}

func TestCommandManagerFIFOFirstWaitsAndLaterCommandsQueue(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 12, 30, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "FIFO", now)
	var wakeups []string
	manager := NewCommandManager(db, func(_ context.Context, deviceKey string) error {
		wakeups = append(wakeups, deviceKey)
		return nil
	}, WithCommandManagerNow(func() time.Time { return now }))

	first, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("submit first command: %v", err)
	}
	second, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("submit second command: %v", err)
	}
	if first.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("first status = %q, want WAITING_DEVICE", first.Status)
	}
	if second.Status != model.CommandStatusQueued {
		t.Fatalf("second status = %q, want QUEUED", second.Status)
	}

	var commands []model.Command
	if err := db.Order("created_at ASC").Order("command_id ASC").Find(&commands).Error; err != nil {
		t.Fatalf("load commands: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("commands = %d, want 2", len(commands))
	}
	byID := map[string]model.Command{commands[0].CommandID: commands[0], commands[1].CommandID: commands[1]}
	if byID[first.CommandID].WaitingAt == nil || byID[first.CommandID].PhaseDeadlineAt == nil {
		t.Fatalf("first command did not enter waiting phase: %#v", byID[first.CommandID])
	}
	if byID[second.CommandID].WaitingAt != nil || byID[second.CommandID].PhaseDeadlineAt != nil {
		t.Fatalf("queued command started a phase deadline: %#v", byID[second.CommandID])
	}
	if want := []string{"001122-FIFO"}; !reflect.DeepEqual(wakeups, want) {
		t.Fatalf("wakeups = %#v, want %#v", wakeups, want)
	}
}

func TestCommandManagerRedisEnqueueFailureTerminatesCreatedCommand(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 13, 0, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "REDIS-FAIL", now)
	injected := errors.New("redis unavailable")
	manager := NewCommandManager(db, func(context.Context, string) error { return injected }, WithCommandManagerNow(func() time.Time { return now }))

	result, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
	if !errors.Is(err, ErrCommandQueueUnavailable) || !errors.Is(err, injected) {
		t.Fatalf("Submit() error = %v, want queue and injected errors", err)
	}
	if result.CommandID == "" || result.Status != model.CommandStatusFailed {
		t.Fatalf("Submit() result = %#v, want stable FAILED result", result)
	}

	var command model.Command
	if err := db.First(&command, "command_id = ?", result.CommandID).Error; err != nil {
		t.Fatalf("load failed command: %v", err)
	}
	if command.Status != model.CommandStatusFailed || command.FailureStage != "redis.enqueue" {
		t.Fatalf("command status/stage = %s/%q, want FAILED/redis.enqueue", command.Status, command.FailureStage)
	}
	if command.FinishedAt == nil || command.PhaseDeadlineAt != nil {
		t.Fatalf("failed command timestamps = finished:%v deadline:%v", command.FinishedAt, command.PhaseDeadlineAt)
	}

	var failureEvent model.CommandEvent
	if err := db.Where("command_id = ? AND stage = ?", result.CommandID, "redis.enqueue").First(&failureEvent).Error; err != nil {
		t.Fatalf("load redis failure event: %v", err)
	}
	if failureEvent.FromStatus != model.CommandStatusWaitingDevice || failureEvent.ToStatus != model.CommandStatusFailed {
		t.Fatalf("redis failure transition = %s -> %s", failureEvent.FromStatus, failureEvent.ToStatus)
	}
}

func TestCommandManagerTransfersReceiveUniqueSystemCommandKeys(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 13, 30, 0, 0, time.UTC)
	downloadDevice := createCommandManagerDevice(t, db, "DOWNLOAD", now)
	uploadDevice := createCommandManagerDevice(t, db, "UPLOAD", now)
	setCommandManagerCapabilities(t, db, downloadDevice.ID, `["Download"]`)
	setCommandManagerCapabilities(t, db, uploadDevice.ID, `["Upload"]`)
	manager := NewCommandManager(db, func(context.Context, string) error { return nil }, WithCommandManagerNow(func() time.Time { return now }))

	downloadRequest := req.DownloadRequest{FileType: "1 Firmware Upgrade Image", URL: "https://example.com/fw.bin", FileSize: 1024}
	uploadRequest := req.UploadRequest{FileType: "1 Vendor Configuration File", URL: "https://example.com/config.xml"}
	download, err := manager.Submit(context.Background(), downloadDevice.ID, "Download", downloadRequest)
	if err != nil {
		t.Fatalf("submit Download: %v", err)
	}
	upload, err := manager.Submit(context.Background(), uploadDevice.ID, "Upload", uploadRequest)
	if err != nil {
		t.Fatalf("submit Upload: %v", err)
	}

	var commands []model.Command
	if err := db.Where("command_id IN ?", []string{download.CommandID, upload.CommandID}).Find(&commands).Error; err != nil {
		t.Fatalf("load transfer commands: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("transfer commands = %d, want 2", len(commands))
	}
	keys := make(map[string]struct{}, len(commands))
	for _, command := range commands {
		if command.CommandKey == nil || !strings.HasPrefix(*command.CommandKey, "rpc-") {
			t.Fatalf("%s command key = %v, want rpc- prefix", command.Operation, command.CommandKey)
		}
		keys[*command.CommandKey] = struct{}{}
		if strings.Contains(string(command.ParamsJSON), "commandKey") {
			t.Fatalf("%s persisted system CommandKey in typed params: %s", command.Operation, command.ParamsJSON)
		}
	}
	if len(keys) != 2 {
		t.Fatalf("unique system command keys = %d, want 2", len(keys))
	}
}

func TestCommandManagerRetryCopiesTypedParamsAndPreservesOriginal(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "RETRY", now)
	setCommandManagerCapabilities(t, db, device.ID, `["GetParameterValues"]`)
	request := req.GetParameterValuesRequest{Paths: []string{"Device.DeviceInfo.SerialNumber", "Device.DeviceInfo.SoftwareVersion"}}
	paramsJSON, err := EncodeRPCRequest("GetParameterValues", request)
	if err != nil {
		t.Fatalf("encode original request: %v", err)
	}
	finishedAt := now.Add(-time.Minute)
	original := model.Command{
		CommandID:    "original-failed-command",
		DeviceID:     device.ID,
		DeviceKey:    "001122-RETRY",
		Operation:    "GetParameterValues",
		ParamsJSON:   model.LongTextJSON(paramsJSON),
		Status:       model.CommandStatusFailed,
		QueuedAt:     now.Add(-2 * time.Minute),
		FinishedAt:   &finishedAt,
		FailureStage: "cwmp.fault",
		FaultCode:    9002,
		FaultString:  "internal error",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    finishedAt,
	}
	if err := NewCommandStore(db).Create(context.Background(), &original); err != nil {
		t.Fatalf("seed original command: %v", err)
	}

	manager := NewCommandManager(db, func(context.Context, string) error { return nil }, WithCommandManagerNow(func() time.Time { return now }))
	result, err := manager.Retry(context.Background(), original.CommandID)
	if err != nil {
		t.Fatalf("Retry() error: %v", err)
	}
	if result.CommandID == "" || result.CommandID == original.CommandID || result.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("Retry() result = %#v", result)
	}

	var retried model.Command
	if err := db.First(&retried, "command_id = ?", result.CommandID).Error; err != nil {
		t.Fatalf("load retried command: %v", err)
	}
	if retried.RetryOf != original.CommandID {
		t.Fatalf("retryOf = %q, want %q", retried.RetryOf, original.CommandID)
	}
	if string(retried.ParamsJSON) != string(original.ParamsJSON) {
		t.Fatalf("retried params = %s, want exact original %s", retried.ParamsJSON, original.ParamsJSON)
	}

	var preserved model.Command
	if err := db.First(&preserved, "command_id = ?", original.CommandID).Error; err != nil {
		t.Fatalf("reload original command: %v", err)
	}
	if preserved.Status != model.CommandStatusFailed || preserved.FinishedAt == nil || !preserved.FinishedAt.Equal(finishedAt) || preserved.FaultCode != original.FaultCode || preserved.FaultString != original.FaultString {
		t.Fatalf("original command changed: %#v", preserved)
	}
}

func TestCommandManagerFIFOWaitingTransferBlocksNextCommand(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 14, 30, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "TRANSFER-BLOCK", now)
	commandKey := "rpc-blocking-transfer"
	head := model.Command{
		CommandID: "blocking-transfer", DeviceID: device.ID, DeviceKey: "001122-TRANSFER-BLOCK", Operation: "Download",
		ParamsJSON: model.LongTextJSON(`{"fileType":"1 Firmware Upgrade Image","url":"https://example.com/fw.bin","username":"","password":"","fileSize":1,"targetFileName":"","delaySeconds":0,"successURL":"","failureURL":""}`),
		CommandKey: &commandKey, Status: model.CommandStatusWaitingTransfer, QueuedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
	}
	if err := NewCommandStore(db).Create(context.Background(), &head); err != nil {
		t.Fatalf("seed WAITING_TRANSFER head: %v", err)
	}
	wakeups := 0
	manager := NewCommandManager(db, func(context.Context, string) error { wakeups++; return nil }, WithCommandManagerNow(func() time.Time { return now }))
	result, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("submit command behind transfer: %v", err)
	}
	if result.Status != model.CommandStatusQueued || wakeups != 0 {
		t.Fatalf("command behind transfer = status:%s wakeups:%d, want QUEUED/0", result.Status, wakeups)
	}
}

func TestCommandManagerDifferentDeviceHeadsDispatchIndependently(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC)
	firstDevice := createCommandManagerDevice(t, db, "PARALLEL-A", now)
	secondDevice := createCommandManagerDevice(t, db, "PARALLEL-B", now)
	var wakeups []string
	manager := NewCommandManager(db, func(_ context.Context, deviceKey string) error {
		wakeups = append(wakeups, deviceKey)
		return nil
	}, WithCommandManagerNow(func() time.Time { return now }))

	first, err := manager.Submit(context.Background(), firstDevice.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("submit first device: %v", err)
	}
	second, err := manager.Submit(context.Background(), secondDevice.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("submit second device: %v", err)
	}
	if first.Status != model.CommandStatusWaitingDevice || second.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("cross-device statuses = %s/%s, want independent WAITING_DEVICE heads", first.Status, second.Status)
	}
	wantWakeups := []string{"001122-PARALLEL-A", "001122-PARALLEL-B"}
	if !reflect.DeepEqual(wakeups, wantWakeups) {
		t.Fatalf("cross-device wakeups = %#v, want %#v", wakeups, wantWakeups)
	}
}

func TestCommandManagerLocksDeviceRowForFIFOHeadDecision(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 16, 15, 30, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "ROW-LOCK", now)
	lockClauseObserved := false
	if err := db.Callback().Query().Before("gorm:query").Register("test:observe_device_update_lock", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "Device" {
			if _, ok := tx.Statement.Clauses["FOR"]; ok {
				lockClauseObserved = true
			}
		}
	}); err != nil {
		t.Fatalf("register row-lock observer: %v", err)
	}
	manager := NewCommandManager(db, func(context.Context, string) error { return nil }, WithCommandManagerNow(func() time.Time { return now }))
	if _, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil); err != nil {
		t.Fatalf("Submit() error: %v", err)
	}
	if !lockClauseObserved {
		t.Fatal("device SELECT did not carry clause.Locking UPDATE")
	}
}
