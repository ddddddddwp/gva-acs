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

func newGormCommandRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate command repo models: %v", err)
	}
	return db
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
	if sent.Status != model.CommandStatusSent || sent.RequestID != "cwmp-request-20" || sent.SentAt == nil || !sent.SentAt.Equal(sentAt) {
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
	if preserved.Status != model.CommandStatusBuilding || preserved.RequestID != "" || preserved.SentAt != nil || preserved.Version != 0 {
		t.Fatalf("command changed despite event rollback: %#v", preserved)
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
