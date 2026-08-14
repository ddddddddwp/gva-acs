package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newCommandQueueAdvancerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate queue advancer models: %v", err)
	}
	return db
}

func seedQueueAdvancerDevice(t *testing.T, db *gorm.DB, serial string) model.Device {
	t.Helper()
	device := model.Device{OUI: "8CE468", SerialNumber: serial}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	return device
}

func seedQueueAdvancerCommand(t *testing.T, db *gorm.DB, device model.Device, commandID, status string, createdAt time.Time) {
	t.Helper()
	command := model.Command{
		CommandID:  commandID,
		DeviceID:   device.ID,
		DeviceKey:  device.OUI + "-" + device.SerialNumber,
		Operation:  "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     status,
		QueuedAt:   createdAt,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	if err := NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed command %s: %v", commandID, err)
	}
}

func TestCommandQueueAdvancerPromotesOnlyOldestQueuedCommand(t *testing.T) {
	db := newCommandQueueAdvancerTestDB(t)
	device := seedQueueAdvancerDevice(t, db, "FIFO")
	base := time.Date(2026, 7, 19, 17, 0, 0, 0, time.UTC)
	seedQueueAdvancerCommand(t, db, device, "completed-head", model.CommandStatusCompleted, base)
	seedQueueAdvancerCommand(t, db, device, "queued-first", model.CommandStatusQueued, base.Add(time.Second))
	seedQueueAdvancerCommand(t, db, device, "queued-second", model.CommandStatusQueued, base.Add(2*time.Second))

	wakeups := make([]string, 0, 1)
	advanceAt := base.Add(time.Minute)
	advancer := NewCommandQueueAdvancer(db, func(_ context.Context, deviceKey string) error {
		wakeups = append(wakeups, deviceKey)
		return nil
	}, WithCommandQueueAdvancerNow(func() time.Time { return advanceAt }))

	result, err := advancer.AdvanceDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("AdvanceDevice: %v", err)
	}
	if !result.Promoted || result.CommandID != "queued-first" {
		t.Fatalf("advance result = %#v", result)
	}
	if want := []string{"8CE468-FIFO"}; !reflect.DeepEqual(wakeups, want) {
		t.Fatalf("wakeups = %#v, want %#v", wakeups, want)
	}

	var first, second model.Command
	if err := db.First(&first, "command_id = ?", "queued-first").Error; err != nil {
		t.Fatalf("load first queued command: %v", err)
	}
	if err := db.First(&second, "command_id = ?", "queued-second").Error; err != nil {
		t.Fatalf("load second queued command: %v", err)
	}
	if first.Status != model.CommandStatusWaitingDevice || first.WaitingAt == nil || !first.WaitingAt.Equal(advanceAt) {
		t.Fatalf("promoted command = %#v", first)
	}
	wantDeadline := advanceAt.Add(config.CurrentRuntime().CommandQueueWaitTimeout)
	if first.PhaseDeadlineAt == nil || !first.PhaseDeadlineAt.Equal(wantDeadline) {
		t.Fatalf("deadline = %v, want %v", first.PhaseDeadlineAt, wantDeadline)
	}
	if second.Status != model.CommandStatusQueued || second.WaitingAt != nil || second.PhaseDeadlineAt != nil {
		t.Fatalf("later command = %#v", second)
	}
	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", first.CommandID, "QUEUE_ADVANCED").First(&event).Error; err != nil {
		t.Fatalf("load queue event: %v", err)
	}
	if event.FromStatus != model.CommandStatusQueued || event.ToStatus != model.CommandStatusWaitingDevice {
		t.Fatalf("queue event = %#v", event)
	}

	secondResult, err := advancer.AdvanceDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("second AdvanceDevice: %v", err)
	}
	if secondResult.Promoted || len(wakeups) != 1 {
		t.Fatalf("duplicate advance result=%#v wakeups=%#v", secondResult, wakeups)
	}
}

func TestCommandQueueAdvancerDoesNotPassActiveHead(t *testing.T) {
	db := newCommandQueueAdvancerTestDB(t)
	device := seedQueueAdvancerDevice(t, db, "ACTIVE")
	base := time.Date(2026, 7, 19, 17, 10, 0, 0, time.UTC)
	seedQueueAdvancerCommand(t, db, device, "sent-head", model.CommandStatusSent, base)
	seedQueueAdvancerCommand(t, db, device, "queued-behind", model.CommandStatusQueued, base.Add(time.Second))
	wakeups := 0
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups++
		return nil
	})

	result, err := advancer.AdvanceDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("AdvanceDevice: %v", err)
	}
	if result.Promoted || wakeups != 0 {
		t.Fatalf("result=%#v wakeups=%d", result, wakeups)
	}
	var queued model.Command
	if err := db.First(&queued, "command_id = ?", "queued-behind").Error; err != nil {
		t.Fatalf("load queued command: %v", err)
	}
	if queued.Status != model.CommandStatusQueued {
		t.Fatalf("queued status = %s", queued.Status)
	}
}

func TestCommandQueueAdvancerKeepsPromotedCommandWhenWakeupFails(t *testing.T) {
	db := newCommandQueueAdvancerTestDB(t)
	device := seedQueueAdvancerDevice(t, db, "WAKE-FAIL")
	now := time.Date(2026, 7, 19, 17, 20, 0, 0, time.UTC)
	seedQueueAdvancerCommand(t, db, device, "queued-wake-fail", model.CommandStatusQueued, now)
	injected := errors.New("redis unavailable")
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error { return injected })

	result, err := advancer.AdvanceDevice(context.Background(), device.ID)
	if !errors.Is(err, injected) {
		t.Fatalf("AdvanceDevice error = %v, want %v", err, injected)
	}
	if !result.Promoted || result.CommandID != "queued-wake-fail" {
		t.Fatalf("advance result = %#v", result)
	}
	var command model.Command
	if err := db.First(&command, "command_id = ?", result.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if command.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("status after Redis failure = %s", command.Status)
	}
	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", result.CommandID, "WAKE_ENQUEUE_FAILED").First(&event).Error; err != nil {
		t.Fatalf("load wake failure event: %v", err)
	}
	if event.Message != injected.Error() {
		t.Fatalf("wake failure event = %#v", event)
	}
}

func TestCommandQueueAdvancerRecoversOrphanAndWaitingHeads(t *testing.T) {
	db := newCommandQueueAdvancerTestDB(t)
	base := time.Date(2026, 7, 19, 17, 30, 0, 0, time.UTC)
	orphan := seedQueueAdvancerDevice(t, db, "ORPHAN")
	waiting := seedQueueAdvancerDevice(t, db, "WAITING")
	busy := seedQueueAdvancerDevice(t, db, "BUSY")
	seedQueueAdvancerCommand(t, db, orphan, "orphan-queued", model.CommandStatusQueued, base)
	seedQueueAdvancerCommand(t, db, waiting, "existing-waiting", model.CommandStatusWaitingDevice, base)
	seedQueueAdvancerCommand(t, db, busy, "busy-sent", model.CommandStatusSent, base)
	seedQueueAdvancerCommand(t, db, busy, "busy-queued", model.CommandStatusQueued, base.Add(time.Second))
	wakeups := make([]string, 0, 2)
	advancer := NewCommandQueueAdvancer(db, func(_ context.Context, deviceKey string) error {
		wakeups = append(wakeups, deviceKey)
		return nil
	}, WithCommandQueueAdvancerNow(func() time.Time { return base.Add(time.Minute) }))

	if err := advancer.Recover(context.Background()); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if want := []string{"8CE468-ORPHAN", "8CE468-WAITING"}; !reflect.DeepEqual(wakeups, want) {
		t.Fatalf("wakeups = %#v, want %#v", wakeups, want)
	}
	var orphanCommand, busyQueued model.Command
	if err := db.First(&orphanCommand, "command_id = ?", "orphan-queued").Error; err != nil {
		t.Fatalf("load orphan command: %v", err)
	}
	if err := db.First(&busyQueued, "command_id = ?", "busy-queued").Error; err != nil {
		t.Fatalf("load busy queued command: %v", err)
	}
	if orphanCommand.Status != model.CommandStatusWaitingDevice || busyQueued.Status != model.CommandStatusQueued {
		t.Fatalf("recovered statuses orphan=%s busy=%s", orphanCommand.Status, busyQueued.Status)
	}
}

func TestCommandQueueAdvancerConcurrentCallsPromoteOnce(t *testing.T) {
	db := newCommandQueueAdvancerTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	device := seedQueueAdvancerDevice(t, db, "CONCURRENT")
	now := time.Date(2026, 7, 19, 17, 40, 0, 0, time.UTC)
	seedQueueAdvancerCommand(t, db, device, "concurrent-first", model.CommandStatusQueued, now)
	seedQueueAdvancerCommand(t, db, device, "concurrent-second", model.CommandStatusQueued, now.Add(time.Second))
	var wakeups atomic.Int32
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups.Add(1)
		return nil
	})

	start := make(chan struct{})
	results := make([]CommandQueueAdvanceResult, 2)
	errs := make([]error, 2)
	var group sync.WaitGroup
	for i := range results {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			results[index], errs[index] = advancer.AdvanceDevice(context.Background(), device.ID)
		}(i)
	}
	close(start)
	group.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("concurrent AdvanceDevice: %v", err)
		}
	}
	promotions := 0
	for _, result := range results {
		if result.Promoted {
			promotions++
		}
	}
	if promotions != 1 || wakeups.Load() != 1 {
		t.Fatalf("promotions=%d wakeups=%d results=%#v", promotions, wakeups.Load(), results)
	}
	var waiting, queued int64
	db.Model(new(model.Command)).Where("device_id = ? AND status = ?", device.ID, model.CommandStatusWaitingDevice).Count(&waiting)
	db.Model(new(model.Command)).Where("device_id = ? AND status = ?", device.ID, model.CommandStatusQueued).Count(&queued)
	if waiting != 1 || queued != 1 {
		t.Fatalf("waiting=%d queued=%d", waiting, queued)
	}
}
