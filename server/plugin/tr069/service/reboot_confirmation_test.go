package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newRebootLifecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate reboot lifecycle models: %v", err)
	}
	return db
}

func TestRebootConfirmationAdvancesNextFIFOCommand(t *testing.T) {
	db := newRebootLifecycleTestDB(t)
	now := time.Date(2026, 7, 19, 18, 10, 0, 0, time.UTC)
	device := model.Device{OUI: "8CE468", SerialNumber: "REBOOT-NEXT"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	reboot := seedWaitingReboot(t, db, "reboot-before-next", device.ID, now.Add(-time.Minute), now.Add(time.Minute))
	next := model.Command{
		CommandID: "queued-after-reboot", DeviceID: device.ID, DeviceKey: device.OUI + "-" + device.SerialNumber,
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusQueued,
		QueuedAt: now, CreatedAt: now,
	}
	if err := NewCommandStore(db).Create(context.Background(), &next); err != nil {
		t.Fatalf("seed next command: %v", err)
	}
	wakeups := 0
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups++
		return nil
	})
	if err := NewRebootConfirmationService(db, advancer).ConfirmFromInform(context.Background(), device.ID, []string{"1 BOOT"}, now); err != nil {
		t.Fatalf("ConfirmFromInform: %v", err)
	}
	var confirmed, promoted model.Command
	if err := db.First(&confirmed, "command_id = ?", reboot.CommandID).Error; err != nil {
		t.Fatalf("load confirmed command: %v", err)
	}
	if err := db.First(&promoted, "command_id = ?", next.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if confirmed.Status != model.CommandStatusCompleted || promoted.Status != model.CommandStatusWaitingDevice || wakeups != 1 {
		t.Fatalf("statuses confirmed=%s promoted=%s wakeups=%d", confirmed.Status, promoted.Status, wakeups)
	}
}

func seedWaitingReboot(t *testing.T, db *gorm.DB, commandID string, deviceID uint, createdAt, deadline time.Time) model.Command {
	t.Helper()
	command := model.Command{
		CommandID: commandID, DeviceID: deviceID, DeviceKey: "001122-REBOOT",
		Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusWaitingReboot, QueuedAt: createdAt,
		CreatedAt: createdAt, PhaseDeadlineAt: &deadline,
	}
	if err := NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed waiting Reboot: %v", err)
	}
	return command
}

func TestRebootConfirmationCompletesOnlyOnExactBootEvents(t *testing.T) {
	for _, eventCode := range []string{"M Reboot", "1 BOOT", "  m reboot  "} {
		t.Run(strings.TrimSpace(eventCode), func(t *testing.T) {
			db := newRebootLifecycleTestDB(t)
			now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
			command := seedWaitingReboot(t, db, "confirm-"+strings.ReplaceAll(eventCode, " ", "-"), 81, now.Add(-time.Minute), now.Add(time.Minute))
			confirmer := NewRebootConfirmationService(db)

			if err := confirmer.ConfirmFromInform(context.Background(), command.DeviceID, []string{eventCode}, now); err != nil {
				t.Fatalf("ConfirmFromInform: %v", err)
			}
			if err := confirmer.ConfirmFromInform(context.Background(), command.DeviceID, []string{"1 BOOT"}, now.Add(time.Second)); err != nil {
				t.Fatalf("duplicate ConfirmFromInform: %v", err)
			}

			var got model.Command
			if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
				t.Fatalf("load confirmed command: %v", err)
			}
			if got.Status != model.CommandStatusCompleted || got.FinishedAt == nil || got.PhaseDeadlineAt != nil {
				t.Fatalf("confirmed command = %#v", got)
			}
			var events int64
			if err := db.Model(new(model.CommandEvent)).
				Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_CONFIRMED").
				Count(&events).Error; err != nil {
				t.Fatalf("count confirmation events: %v", err)
			}
			if events != 1 {
				t.Fatalf("confirmation events = %d, want 1", events)
			}
		})
	}

	for _, eventCode := range []string{"2 PERIODIC", "4 VALUE CHANGE", "0 BOOTSTRAP", "1 BOOTSTRAP", "M Rebooted"} {
		t.Run("ignore "+eventCode, func(t *testing.T) {
			db := newRebootLifecycleTestDB(t)
			now := time.Now()
			command := seedWaitingReboot(t, db, "ignore-"+strings.ReplaceAll(eventCode, " ", "-"), 82, now, now.Add(time.Minute))
			if err := NewRebootConfirmationService(db).ConfirmFromInform(context.Background(), command.DeviceID, []string{eventCode}, now); err != nil {
				t.Fatalf("ConfirmFromInform: %v", err)
			}
			var got model.Command
			if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
				t.Fatalf("load ignored command: %v", err)
			}
			if got.Status != model.CommandStatusWaitingReboot {
				t.Fatalf("event %q changed status to %s", eventCode, got.Status)
			}
		})
	}
}

func TestRebootTimeoutScannerExpiresOnlyDueWaitingReboots(t *testing.T) {
	db := newRebootLifecycleTestDB(t)
	now := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)
	expired := seedWaitingReboot(t, db, "expired-reboot", 91, now.Add(-time.Minute), now.Add(-time.Second))
	future := seedWaitingReboot(t, db, "future-reboot", 92, now.Add(-time.Minute), now.Add(time.Minute))

	scanner := NewRebootTimeoutScanner(db)
	if err := scanner.ScanOnce(context.Background(), now); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	if err := scanner.ScanOnce(context.Background(), now.Add(time.Second)); err != nil {
		t.Fatalf("second ScanOnce: %v", err)
	}

	var expiredGot, futureGot model.Command
	if err := db.First(&expiredGot, "command_id = ?", expired.CommandID).Error; err != nil {
		t.Fatalf("load expired command: %v", err)
	}
	if err := db.First(&futureGot, "command_id = ?", future.CommandID).Error; err != nil {
		t.Fatalf("load future command: %v", err)
	}
	if expiredGot.Status != model.CommandStatusTimeout || expiredGot.FinishedAt == nil || expiredGot.PhaseDeadlineAt != nil {
		t.Fatalf("expired command = %#v", expiredGot)
	}
	if futureGot.Status != model.CommandStatusWaitingReboot {
		t.Fatalf("future command status = %s", futureGot.Status)
	}
	var events int64
	if err := db.Model(new(model.CommandEvent)).
		Where("command_id = ? AND event_type = ?", expired.CommandID, "REBOOT_CONFIRM_TIMEOUT").
		Count(&events).Error; err != nil {
		t.Fatalf("count timeout events: %v", err)
	}
	if events != 1 {
		t.Fatalf("timeout events = %d, want 1", events)
	}
}

func TestRebootTimeoutAdvancesNextFIFOCommand(t *testing.T) {
	db := newRebootLifecycleTestDB(t)
	now := time.Date(2026, 7, 19, 18, 30, 0, 0, time.UTC)
	device := model.Device{OUI: "8CE468", SerialNumber: "TIMEOUT-NEXT"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	seedWaitingReboot(t, db, "expired-before-next", device.ID, now.Add(-time.Minute), now.Add(-time.Second))
	next := model.Command{
		CommandID: "queued-after-timeout", DeviceID: device.ID, DeviceKey: device.OUI + "-" + device.SerialNumber,
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusQueued,
		QueuedAt: now, CreatedAt: now,
	}
	if err := NewCommandStore(db).Create(context.Background(), &next); err != nil {
		t.Fatalf("seed queued command: %v", err)
	}
	wakeups := 0
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups++
		return nil
	})
	if err := NewRebootTimeoutScanner(db, advancer).ScanOnce(context.Background(), now); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	var promoted model.Command
	if err := db.First(&promoted, "command_id = ?", next.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if promoted.Status != model.CommandStatusWaitingDevice || wakeups != 1 {
		t.Fatalf("promoted status=%s wakeups=%d", promoted.Status, wakeups)
	}
}
