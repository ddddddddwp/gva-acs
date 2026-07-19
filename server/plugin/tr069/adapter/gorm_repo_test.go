package adapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	tr069core "github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type recordingUploadIdentityBinder struct {
	bindings []UploadIdentityBinding
	ttls     []time.Duration
	err      error
}

func (b *recordingUploadIdentityBinder) Bind(_ context.Context, binding UploadIdentityBinding, ttl time.Duration) error {
	b.bindings = append(b.bindings, binding)
	b.ttls = append(b.ttls, ttl)
	return b.err
}

func TestGormDeviceRepoBindsUploadIdentityAfterSuccessfulInform(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate device models: %v", err)
	}
	binder := new(recordingUploadIdentityBinder)
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	repo.SetUploadIdentityBinder(binder)
	info := &tr069core.InformSummary{Params: map[string]string{"Device.DeviceInfo.SerialNumber": "INFORM-BIND"}}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.55"); err != nil {
		t.Fatalf("upsert Inform: %v", err)
	}
	if len(binder.bindings) != 1 || binder.bindings[0].DeviceID == 0 || binder.bindings[0].SerialNumber != "INFORM-BIND" || binder.bindings[0].IP != "192.0.2.55" {
		t.Fatalf("bindings=%#v", binder.bindings)
	}
	if len(binder.ttls) != 1 || binder.ttls[0] <= 0 {
		t.Fatalf("binding TTLs=%#v", binder.ttls)
	}
}

func TestGormDeviceRepoDoesNotBindWhenInformPersistenceFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	binder := new(recordingUploadIdentityBinder)
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	repo.SetUploadIdentityBinder(binder)
	info := &tr069core.InformSummary{Params: map[string]string{"Device.DeviceInfo.SerialNumber": "INFORM-FAIL"}}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.56"); err == nil {
		t.Fatal("Inform persistence unexpectedly succeeded without device table")
	}
	if len(binder.bindings) != 0 {
		t.Fatalf("binding written after persistence failure: %#v", binder.bindings)
	}
}

func TestGormDeviceRepoRejectsInformForDeletingDevice(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate device models: %v", err)
	}
	lastInform := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	deletingAt := lastInform.Add(time.Minute)
	device := model.Device{OUI: "001122", SerialNumber: "INFORM-DELETING", IP: "192.0.2.80", LastInform: lastInform, DeletingAt: &deletingAt}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create deleting device: %v", err)
	}
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	info := &tr069core.InformSummary{Params: map[string]string{
		"Device.DeviceInfo.SerialNumber":    "INFORM-DELETING",
		"Device.DeviceInfo.SoftwareVersion": "must-not-update",
	}}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.81"); !errors.Is(err, service.ErrDeviceDeleting) {
		t.Fatalf("UpsertFromInform() error = %v, want ErrDeviceDeleting", err)
	}
	var kept model.Device
	if err := db.First(&kept, device.ID).Error; err != nil {
		t.Fatalf("load deleting device: %v", err)
	}
	if !kept.LastInform.Equal(lastInform) || kept.IP != device.IP || kept.SoftwareVer != "" {
		t.Fatalf("deleting device was updated: %#v", kept)
	}
	var values int64
	if err := db.Model(new(model.DataModelValue)).Where("device_id = ?", device.ID).Count(&values).Error; err != nil || values != 0 {
		t.Fatalf("parameter count = %d, error = %v", values, err)
	}
}

func newGormCommandRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandEvent), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate command repo models: %v", err)
	}
	return db
}

func TestGormCommandRepoCompletionAdvancesNextFIFOCommand(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	if err := db.AutoMigrate(new(model.Device)); err != nil {
		t.Fatalf("migrate device: %v", err)
	}
	now := time.Date(2026, 7, 19, 18, 0, 0, 0, time.UTC)
	device := model.Device{OUI: "8CE468", SerialNumber: "AUTO-NEXT"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	current := model.Command{
		CommandID: "current-sent", DeviceID: device.ID, DeviceKey: "8CE468-AUTO-NEXT",
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
	}
	next := model.Command{
		CommandID: "next-queued", DeviceID: device.ID, DeviceKey: "8CE468-AUTO-NEXT",
		Operation: "GetParameterValues", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusQueued, QueuedAt: now.Add(time.Second), CreatedAt: now.Add(time.Second),
	}
	for _, command := range []*model.Command{&current, &next} {
		if err := service.NewCommandStore(db).Create(context.Background(), command); err != nil {
			t.Fatalf("seed command %s: %v", command.CommandID, err)
		}
	}
	wakeups := make([]string, 0, 1)
	advancer := service.NewCommandQueueAdvancer(db, func(_ context.Context, deviceKey string) error {
		wakeups = append(wakeups, deviceKey)
		return nil
	}, service.WithCommandQueueAdvancerNow(func() time.Time { return now.Add(time.Minute) }))

	if err := NewGormCommandRepo(db, advancer).MarkSuccess(context.Background(), current.CommandID, now.Add(30*time.Second)); err != nil {
		t.Fatalf("MarkSuccess: %v", err)
	}
	var completed, promoted model.Command
	if err := db.First(&completed, "command_id = ?", current.CommandID).Error; err != nil {
		t.Fatalf("load completed command: %v", err)
	}
	if err := db.First(&promoted, "command_id = ?", next.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if completed.Status != model.CommandStatusCompleted || promoted.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("statuses completed=%s promoted=%s", completed.Status, promoted.Status)
	}
	if len(wakeups) != 1 || wakeups[0] != next.DeviceKey {
		t.Fatalf("wakeups=%#v", wakeups)
	}
}

func TestGormCommandRepoFailureAdvancesNextFIFOCommand(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	if err := db.AutoMigrate(new(model.Device)); err != nil {
		t.Fatalf("migrate device: %v", err)
	}
	now := time.Date(2026, 7, 19, 18, 5, 0, 0, time.UTC)
	device := model.Device{OUI: "8CE468", SerialNumber: "FAILURE-NEXT"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	current := model.Command{
		CommandID: "current-building-failure", DeviceID: device.ID, DeviceKey: "8CE468-FAILURE-NEXT",
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusBuilding, QueuedAt: now, CreatedAt: now,
	}
	next := model.Command{
		CommandID: "next-after-failure", DeviceID: device.ID, DeviceKey: current.DeviceKey,
		Operation: "GetParameterValues", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusQueued, QueuedAt: now.Add(time.Second), CreatedAt: now.Add(time.Second),
	}
	for _, command := range []*model.Command{&current, &next} {
		if err := service.NewCommandStore(db).Create(context.Background(), command); err != nil {
			t.Fatalf("seed command %s: %v", command.CommandID, err)
		}
	}
	wakeups := 0
	advancer := service.NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups++
		return nil
	})
	if err := NewGormCommandRepo(db, advancer).MarkFail(context.Background(), current.CommandID, 9002, "fault", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkFail: %v", err)
	}
	var promoted model.Command
	if err := db.First(&promoted, "command_id = ?", next.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if promoted.Status != model.CommandStatusWaitingDevice || wakeups != 1 {
		t.Fatalf("promoted status=%s wakeups=%d", promoted.Status, wakeups)
	}
}

func TestGormDeviceRepoProvisionerSchedulesNeededInformProfile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate device repo models: %v", err)
	}
	cipher, err := NewCredentialCipher(testCredentialConfig(0x63))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	scheduler := &recordingCredentialScheduler{accept: true}
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, cipher), scheduler)
	info := &tr069core.InformSummary{Params: map[string]string{
		"Device.DeviceInfo.SerialNumber":               "INFORM-SCHEDULE",
		"Device.ManagementServer.ConnectionRequestURL": "http://127.0.0.1:8400",
	}}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.20"); err != nil {
		t.Fatalf("Inform: %v", err)
	}
	var device model.Device
	if err := db.First(&device, "serial_number = ?", "INFORM-SCHEDULE").Error; err != nil {
		t.Fatalf("load device: %v", err)
	}
	if len(scheduler.deviceIDs) != 1 || scheduler.deviceIDs[0] != device.ID {
		t.Fatalf("scheduled device IDs = %#v, want [%d]", scheduler.deviceIDs, device.ID)
	}
}

func TestGormDeviceRepoInformDoesNotClearConnectionRequestURL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate device repo models: %v", err)
	}
	cipher, err := NewCredentialCipher(testCredentialConfig(0x61))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	profiles := NewConnectionProfileRepository(db, cipher)
	repo := NewGormDeviceRepo(db, profiles)

	first := &tr069core.InformSummary{Params: map[string]string{
		"Device.DeviceInfo.SerialNumber":               "INFORM-PRESERVE",
		"Device.ManagementServer.ConnectionRequestURL": "http://127.0.0.1:8400",
	}}
	if _, err := repo.UpsertFromInform(context.Background(), first, "192.0.2.10"); err != nil {
		t.Fatalf("first Inform: %v", err)
	}
	second := &tr069core.InformSummary{Params: map[string]string{
		"Device.DeviceInfo.SerialNumber":    "INFORM-PRESERVE",
		"Device.DeviceInfo.SoftwareVersion": "2.0.0",
	}}
	if _, err := repo.UpsertFromInform(context.Background(), second, "192.0.2.11"); err != nil {
		t.Fatalf("second Inform: %v", err)
	}

	var device model.Device
	if err := db.First(&device, "serial_number = ?", "INFORM-PRESERVE").Error; err != nil {
		t.Fatalf("load device: %v", err)
	}
	if device.ConnectionReqURL != "http://127.0.0.1:8400" {
		t.Fatalf("device connection request URL=%q", device.ConnectionReqURL)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.DiscoveredURL != device.ConnectionReqURL {
		t.Fatalf("profile URL=%q, device URL=%q", profile.DiscoveredURL, device.ConnectionReqURL)
	}
}

func TestGormCommandRepoMarkSendingTransitionsAndAppendsEventTransactionally(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	previousRuntime := config.CurrentRuntime()
	settings := previousRuntime.Settings
	settings.RPCResponseTimeout = 17
	config.StoreRuntime(settings)
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })

	createdAt := time.Date(2026, 7, 16, 15, 30, 0, 0, time.UTC)
	command := model.Command{
		CommandID:  "mark-sending",
		DeviceID:   20,
		DeviceKey:  "001122-SENDING",
		Operation:  "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusBuilding,
		QueuedAt:   createdAt.Add(-time.Minute),
		CreatedAt:  createdAt.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed BUILDING command: %v", err)
	}

	repo := newGormCommandRepo(db)
	sentAt := createdAt.Add(time.Second)
	if err := repo.MarkSending(context.Background(), command.CommandID, "cwmp-request-20", sentAt); err != nil {
		t.Fatalf("MarkSending() error: %v", err)
	}

	var sent model.Command
	if err := db.First(&sent, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load SENT command: %v", err)
	}
	if sent.Status != model.CommandStatusSent || sent.CWMPID != "cwmp-request-20" || sent.SentAt == nil || !sent.SentAt.Equal(sentAt) {
		t.Fatalf("sent command = %#v", sent)
	}
	wantDeadline := sentAt.Add(17 * time.Second)
	if sent.PhaseDeadlineAt == nil || !sent.PhaseDeadlineAt.Equal(wantDeadline) {
		t.Fatalf("phase deadline = %v, want %v", sent.PhaseDeadlineAt, wantDeadline)
	}

	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", command.CommandID, "REQUEST_SENT").First(&event).Error; err != nil {
		t.Fatalf("load REQUEST_SENT event: %v", err)
	}
	if event.FromStatus != model.CommandStatusBuilding || event.ToStatus != model.CommandStatusSent {
		t.Fatalf("REQUEST_SENT transition = %s -> %s", event.FromStatus, event.ToStatus)
	}
}

func TestGormCommandRepoMarkSendingRollsBackWhenEventInsertFails(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	now := time.Now()
	command := model.Command{
		CommandID: "mark-sending-rollback", DeviceID: 21, DeviceKey: "001122-ROLLBACK",
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusBuilding,
		QueuedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed BUILDING command: %v", err)
	}
	injected := errors.New("injected REQUEST_SENT event failure")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_request_sent_event", func(tx *gorm.DB) {
		if event, ok := tx.Statement.Dest.(*model.CommandEvent); ok && event.EventType == "REQUEST_SENT" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register event failure: %v", err)
	}

	repo := newGormCommandRepo(db)
	if err := repo.MarkSending(context.Background(), command.CommandID, "cwmp-rollback", now); !errors.Is(err, injected) {
		t.Fatalf("MarkSending() error = %v, want %v", err, injected)
	}
	var preserved model.Command
	if err := db.First(&preserved, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("reload command: %v", err)
	}
	if preserved.Status != model.CommandStatusBuilding || preserved.CWMPID != "" || preserved.SentAt != nil || preserved.Version != 0 {
		t.Fatalf("command changed despite event rollback: %#v", preserved)
	}
}

func TestGormCommandRepoRebootResponseWaitsForBootInform(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	previous := config.CurrentRuntime()
	settings := previous.Settings
	settings.RebootConfirmTimeout = 41
	config.StoreRuntime(settings)
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })

	acknowledgedAt := time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC)
	command := model.Command{
		CommandID: "reboot-ack", DeviceID: 70, DeviceKey: "001122-REBOOT-ACK",
		Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusSent, QueuedAt: acknowledgedAt.Add(-time.Minute),
		CreatedAt: acknowledgedAt.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed Reboot: %v", err)
	}
	if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, acknowledgedAt); err != nil {
		t.Fatalf("MarkSuccess(Reboot): %v", err)
	}
	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load Reboot: %v", err)
	}
	if got.Status != model.CommandStatusWaitingReboot || got.FinishedAt != nil {
		t.Fatalf("Reboot response finalized command: %#v", got)
	}
	wantDeadline := acknowledgedAt.Add(41 * time.Second)
	if got.PhaseDeadlineAt == nil || !got.PhaseDeadlineAt.Equal(wantDeadline) {
		t.Fatalf("deadline = %v, want %v", got.PhaseDeadlineAt, wantDeadline)
	}
	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_ACKNOWLEDGED").First(&event).Error; err != nil {
		t.Fatalf("load acknowledgement event: %v", err)
	}
	if event.Stage != "reboot.acknowledged" ||
		event.FromStatus != model.CommandStatusSent ||
		event.ToStatus != model.CommandStatusWaitingReboot {
		t.Fatalf("acknowledgement event = %#v", event)
	}
}

func TestGormCommandRepoMarkFailAcceptsWaitingReboot(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	failedAt := time.Date(2026, 7, 18, 11, 5, 0, 0, time.UTC)
	deadline := failedAt.Add(time.Minute)
	command := model.Command{
		CommandID: "reboot-fail", DeviceID: 71, DeviceKey: "001122-REBOOT-FAIL",
		Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusWaitingReboot, QueuedAt: failedAt.Add(-time.Minute),
		PhaseDeadlineAt: &deadline, CreatedAt: failedAt.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed WAITING_REBOOT: %v", err)
	}
	if err := newGormCommandRepo(db).MarkFail(context.Background(), command.CommandID, 9002, "reboot failed", failedAt); err != nil {
		t.Fatalf("MarkFail(WAITING_REBOOT): %v", err)
	}
	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load failed Reboot: %v", err)
	}
	if got.Status != model.CommandStatusFailed || got.FinishedAt == nil || !got.FinishedAt.Equal(failedAt) || got.PhaseDeadlineAt != nil {
		t.Fatalf("failed Reboot = %#v", got)
	}
}

func TestGormCommandRepoTerminalCallbacksUseStoreAndNeverRecreateMissingCommands(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	repo := newGormCommandRepo(db)
	now := time.Now()
	missingIDs := []string{"missing-success", "missing-fail"}
	if err := repo.MarkSuccess(context.Background(), missingIDs[0], now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("MarkSuccess(missing) error = %v, want record not found", err)
	}
	if err := repo.MarkFail(context.Background(), missingIDs[1], 9000, "missing", now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("MarkFail(missing) error = %v, want record not found", err)
	}
	var missingCount int64
	if err := db.Model(new(model.Command)).Where("command_id IN ?", missingIDs).Count(&missingCount).Error; err != nil {
		t.Fatalf("count missing callback rows: %v", err)
	}
	if missingCount != 0 {
		t.Fatalf("callbacks recreated %d missing commands", missingCount)
	}

	sent := model.Command{
		CommandID: "mark-success", DeviceID: 22, DeviceKey: "001122-SUCCESS", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusSent, QueuedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
	}
	building := model.Command{
		CommandID: "mark-fail", DeviceID: 23, DeviceKey: "001122-FAIL", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusBuilding, QueuedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
	}
	for _, command := range []*model.Command{&sent, &building} {
		if err := service.NewCommandStore(db).Create(context.Background(), command); err != nil {
			t.Fatalf("seed %s: %v", command.CommandID, err)
		}
	}
	if err := repo.MarkSuccess(context.Background(), sent.CommandID, now); err != nil {
		t.Fatalf("MarkSuccess() error: %v", err)
	}
	if err := repo.MarkFail(context.Background(), building.CommandID, 9003, "invalid arguments", now); err != nil {
		t.Fatalf("MarkFail() error: %v", err)
	}
	var completed, failed model.Command
	if err := db.First(&completed, "command_id = ?", sent.CommandID).Error; err != nil {
		t.Fatalf("load completed command: %v", err)
	}
	if err := db.First(&failed, "command_id = ?", building.CommandID).Error; err != nil {
		t.Fatalf("load failed command: %v", err)
	}
	if completed.Status != model.CommandStatusCompleted {
		t.Fatalf("success status = %s, want COMPLETED", completed.Status)
	}
	if failed.Status != model.CommandStatusFailed || failed.FailureStage != "core.build" || failed.FaultCode != 9003 {
		t.Fatalf("failed callback command = %#v", failed)
	}
}

func TestGormCommandRepoPersistsExplicitQueueAckFailureStage(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	now := time.Now()
	command := model.Command{
		CommandID: "queue-ack-failure", DeviceID: 24, DeviceKey: "001122-ACK", Operation: "Reboot",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed SENT command: %v", err)
	}
	repo := newGormCommandRepo(db)
	if err := repo.MarkFailAtStage(context.Background(), command.CommandID, 9002, "ownership lost", now, tr069core.CommandFailureStageQueueAck); err != nil {
		t.Fatalf("MarkFailAtStage() error: %v", err)
	}
	var failed model.Command
	if err := db.First(&failed, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("reload failed command: %v", err)
	}
	if failed.Status != model.CommandStatusFailed || failed.FailureStage != "queue.ack" {
		t.Fatalf("failed status/stage = %s/%q", failed.Status, failed.FailureStage)
	}
}

func TestProfileTerminalSuccessAndFailureCorrelateOnlyProvisionCommand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newGormCommandRepoTestDB(t)
		now := time.Now()
		command := model.Command{
			CommandID: "profile-terminal-success", DeviceID: 31, DeviceKey: "001122-PROFILE-SUCCESS",
			Operation: "SetParameterValues", Origin: model.CommandOriginSystem, ParamsJSON: model.LongTextJSON(`{}`),
			Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
		}
		if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
			t.Fatalf("seed command: %v", err)
		}
		profile := model.ConnectionProfile{DeviceID: 31, ProvisionState: model.ConnectionProfileStateProvisioning, ProvisionCommandID: command.CommandID, LastError: "old"}
		if err := db.Create(&profile).Error; err != nil {
			t.Fatalf("seed profile: %v", err)
		}
		if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, now); err != nil {
			t.Fatalf("MarkSuccess: %v", err)
		}
		var got model.ConnectionProfile
		if err := db.First(&got, profile.ID).Error; err != nil {
			t.Fatalf("load profile: %v", err)
		}
		if got.ProvisionState != model.ConnectionProfileStateReady || got.LastError != "" {
			t.Fatalf("profile = %#v", got)
		}
	})

	t.Run("failure", func(t *testing.T) {
		db := newGormCommandRepoTestDB(t)
		now := time.Now()
		command := model.Command{
			CommandID: "profile-terminal-failure", DeviceID: 32, DeviceKey: "001122-PROFILE-FAILURE",
			Operation: "SetParameterValues", Origin: model.CommandOriginSystem, ParamsJSON: model.LongTextJSON(`{}`),
			Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
		}
		if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
			t.Fatalf("seed command: %v", err)
		}
		profile := model.ConnectionProfile{DeviceID: 32, ProvisionState: model.ConnectionProfileStateProvisioning, ProvisionCommandID: command.CommandID}
		if err := db.Create(&profile).Error; err != nil {
			t.Fatalf("seed profile: %v", err)
		}
		if err := newGormCommandRepo(db).MarkFail(context.Background(), command.CommandID, 9002, "terminal fault", now); err != nil {
			t.Fatalf("MarkFail: %v", err)
		}
		var got model.ConnectionProfile
		if err := db.First(&got, profile.ID).Error; err != nil {
			t.Fatalf("load profile: %v", err)
		}
		if got.ProvisionState != model.ConnectionProfileStateFailed || got.LastError != "terminal fault" {
			t.Fatalf("profile = %#v", got)
		}
	})

	t.Run("ordinary command", func(t *testing.T) {
		db := newGormCommandRepoTestDB(t)
		now := time.Now()
		command := model.Command{
			CommandID: "ordinary-terminal", DeviceID: 33, DeviceKey: "001122-ORDINARY",
			Operation: "SetParameterValues", Origin: model.CommandOriginUser, ParamsJSON: model.LongTextJSON(`{}`),
			Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
		}
		if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
			t.Fatalf("seed command: %v", err)
		}
		profile := model.ConnectionProfile{DeviceID: 33, ProvisionState: model.ConnectionProfileStateProvisioning, ProvisionCommandID: "some-other-command"}
		if err := db.Create(&profile).Error; err != nil {
			t.Fatalf("seed profile: %v", err)
		}
		if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, now); err != nil {
			t.Fatalf("MarkSuccess: %v", err)
		}
		var got model.ConnectionProfile
		if err := db.First(&got, profile.ID).Error; err != nil {
			t.Fatalf("load profile: %v", err)
		}
		if got.ProvisionState != model.ConnectionProfileStateProvisioning || got.ProvisionCommandID != "some-other-command" {
			t.Fatalf("ordinary command altered profile: %#v", got)
		}
	})

	t.Run("manual credentials supersede an old provision command", func(t *testing.T) {
		db := newGormCommandRepoTestDB(t)
		now := time.Now()
		command := model.Command{
			CommandID: "superseded-profile-terminal", DeviceID: 35, DeviceKey: "001122-SUPERSEDED",
			Operation: "SetParameterValues", Origin: model.CommandOriginSystem, ParamsJSON: model.LongTextJSON(`{}`),
			Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
		}
		if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
			t.Fatalf("seed command: %v", err)
		}
		profile := model.ConnectionProfile{
			DeviceID: 35, ProvisionState: model.ConnectionProfileStateReady,
			ProvisionCommandID: command.CommandID, CredentialSource: model.ConnectionCredentialSourceManual,
		}
		if err := db.Create(&profile).Error; err != nil {
			t.Fatalf("seed profile: %v", err)
		}
		if err := newGormCommandRepo(db).MarkFail(context.Background(), command.CommandID, 9002, "old automatic command failed", now); err != nil {
			t.Fatalf("MarkFail: %v", err)
		}
		var got model.ConnectionProfile
		if err := db.First(&got, profile.ID).Error; err != nil {
			t.Fatalf("load profile: %v", err)
		}
		if got.ProvisionState != model.ConnectionProfileStateReady || got.CredentialSource != model.ConnectionCredentialSourceManual || got.LastError != "" {
			t.Fatalf("old terminal command overrode manual profile: %#v", got)
		}
	})
}

func TestProfileTerminalUpdateFailureRollsBackCommandAndEvent(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	now := time.Now()
	command := model.Command{
		CommandID: "profile-terminal-rollback", DeviceID: 34, DeviceKey: "001122-PROFILE-ROLLBACK",
		Operation: "SetParameterValues", Origin: model.CommandOriginSystem, ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	profile := model.ConnectionProfile{DeviceID: 34, ProvisionState: model.ConnectionProfileStateProvisioning, ProvisionCommandID: command.CommandID}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	injected := errors.New("injected profile terminal update failure")
	if err := db.Callback().Update().Before("gorm:update").Register("test:fail_profile_terminal_update", func(tx *gorm.DB) {
		if tx.Statement.Table == (model.ConnectionProfile{}).TableName() {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register update failure: %v", err)
	}
	if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, now); !errors.Is(err, injected) {
		t.Fatalf("MarkSuccess error = %v, want %v", err, injected)
	}
	var gotCommand model.Command
	if err := db.First(&gotCommand, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if gotCommand.Status != model.CommandStatusSent || gotCommand.Version != 0 {
		t.Fatalf("command changed despite rollback: %#v", gotCommand)
	}
	var terminalEvents int64
	if err := db.Model(new(model.CommandEvent)).Where("command_id = ? AND event_type = ?", command.CommandID, "RESPONSE_COMPLETED").Count(&terminalEvents).Error; err != nil {
		t.Fatalf("count terminal events: %v", err)
	}
	if terminalEvents != 0 {
		t.Fatalf("terminal events after rollback = %d", terminalEvents)
	}
}
