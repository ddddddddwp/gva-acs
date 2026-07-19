package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
)

func seedActiveLifecycleTask(t *testing.T, store *TransferStore, device model.Device, suffix string) model.TransferTask {
	t.Helper()
	now := time.Now().UTC()
	commandID := "lifecycle-command-" + suffix
	commandKey := "lifecycle-command-key-" + suffix
	command := model.Command{
		CommandID: commandID, CommandKey: &commandKey, DeviceID: device.ID, DeviceKey: device.OUI + "-" + device.SerialNumber,
		Operation: "Upload", Status: model.CommandStatusSent, QueuedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	task := model.TransferTask{
		TaskID: "lifecycle-task-" + suffix, DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive,
		CommandID: &commandID, CommandKey: &commandKey, Status: model.TransferStatusWaitingFile, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateActive(context.Background(), &command, &task); err != nil {
		t.Fatalf("create active lifecycle task: %v", err)
	}
	return task
}

func makeLifecycleArtifactAvailable(t *testing.T, store *TransferStore, task model.TransferTask, suffix string, at time.Time) model.Artifact {
	t.Helper()
	current, err := store.FindUniqueWaitingActive(context.Background(), task.DeviceID, task.Channel)
	if err != nil {
		t.Fatalf("find waiting active task: %v", err)
	}
	transitioned, artifact, err := store.CreateActiveReceiving(context.Background(), current, ReceiveMetadata{
		StoragePrefix: "artifacts", Driver: "memory", CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("create active receiving artifact: %v", err)
	}
	_ = transitioned
	available, err := store.MarkArtifactAvailable(context.Background(), artifact.ID, artifact.Version, ArtifactFinalization{Size: 10, SHA256: "sha-" + suffix, ReceivedAt: at})
	if err != nil {
		t.Fatalf("mark artifact available: %v", err)
	}
	return available
}

func TestTransferLifecycleStatusZeroCompletesOnlyAfterFile(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task := seedActiveLifecycleTask(t, store, device, "status-zero")
	lifecycle := NewTransferLifecycle(db)
	at := time.Now().UTC()
	if err := lifecycle.OnUploadResponse(context.Background(), *task.CommandID, 0, at); err != nil {
		t.Fatalf("UploadResponse status 0: %v", err)
	}
	var waiting model.TransferTask
	db.First(&waiting, "task_id = ?", task.TaskID)
	if waiting.Status != model.TransferStatusWaitingFile {
		t.Fatalf("status before file=%s", waiting.Status)
	}
	artifact := makeLifecycleArtifactAvailable(t, store, waiting, "status-zero", at.Add(time.Second))
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ID, at.Add(time.Second)); err != nil {
		t.Fatalf("artifact available: %v", err)
	}
	var completed model.TransferTask
	db.First(&completed, "task_id = ?", task.TaskID)
	if completed.Status != model.TransferStatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("completed task=%#v", completed)
	}
}

func TestTransferLifecycleCompletionAdvancesNextFIFOCommand(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task := seedActiveLifecycleTask(t, store, device, "fifo-next")
	var active model.Command
	if err := db.First(&active, "command_id = ?", *task.CommandID).Error; err != nil {
		t.Fatalf("load active command: %v", err)
	}
	next := model.Command{
		CommandID: "queued-after-transfer", DeviceID: device.ID, DeviceKey: active.DeviceKey,
		Operation: "GetRPCMethods", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusQueued,
		QueuedAt: active.CreatedAt.Add(time.Second), CreatedAt: active.CreatedAt.Add(time.Second),
	}
	if err := NewCommandStore(db).Create(context.Background(), &next); err != nil {
		t.Fatalf("seed queued command: %v", err)
	}
	wakeups := 0
	advancer := NewCommandQueueAdvancer(db, func(context.Context, string) error {
		wakeups++
		return nil
	})
	lifecycle := NewTransferLifecycle(db, advancer)
	at := time.Now().UTC()
	if err := lifecycle.OnUploadResponse(context.Background(), *task.CommandID, 0, at); err != nil {
		t.Fatalf("UploadResponse: %v", err)
	}
	var waiting model.TransferTask
	if err := db.First(&waiting, "task_id = ?", task.TaskID).Error; err != nil {
		t.Fatalf("load waiting task: %v", err)
	}
	artifact := makeLifecycleArtifactAvailable(t, store, waiting, "fifo-next", at.Add(time.Second))
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ID, at.Add(time.Second)); err != nil {
		t.Fatalf("artifact available: %v", err)
	}
	var promoted model.Command
	if err := db.First(&promoted, "command_id = ?", next.CommandID).Error; err != nil {
		t.Fatalf("load promoted command: %v", err)
	}
	if promoted.Status != model.CommandStatusWaitingDevice || wakeups != 1 {
		t.Fatalf("promoted status=%s wakeups=%d", promoted.Status, wakeups)
	}
}

func TestTransferLifecycleStatusOneAcceptsTransferCompleteBeforeFile(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task := seedActiveLifecycleTask(t, store, device, "status-one")
	lifecycle := NewTransferLifecycle(db)
	at := time.Now().UTC()
	if err := lifecycle.OnUploadResponse(context.Background(), *task.CommandID, 1, at); err != nil {
		t.Fatalf("UploadResponse status 1: %v", err)
	}
	var waitingCommand model.Command
	if err := db.First(&waitingCommand, "command_id = ?", *task.CommandID).Error; err != nil || waitingCommand.Status != model.CommandStatusWaitingTransfer || waitingCommand.PhaseDeadlineAt == nil {
		t.Fatalf("waiting command=%#v err=%v", waitingCommand, err)
	}
	if err := lifecycle.OnTransferComplete(context.Background(), *task.CommandKey, 0, "", at.Add(time.Second)); err != nil {
		t.Fatalf("TransferComplete: %v", err)
	}
	var beforeFile model.TransferTask
	db.First(&beforeFile, "task_id = ?", task.TaskID)
	if beforeFile.Status != model.TransferStatusWaitingFile || beforeFile.TransferCompletedAt == nil {
		t.Fatalf("task before file=%#v", beforeFile)
	}
	artifact := makeLifecycleArtifactAvailable(t, store, beforeFile, "status-one", at.Add(2*time.Second))
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ID, at.Add(2*time.Second)); err != nil {
		t.Fatalf("artifact available: %v", err)
	}
	var completed model.TransferTask
	db.First(&completed, "task_id = ?", task.TaskID)
	if completed.Status != model.TransferStatusCompleted {
		t.Fatalf("status after all facts=%s", completed.Status)
	}
}

func TestTransferLifecycleFaultRetainsAvailableArtifactAndIsIdempotent(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task := seedActiveLifecycleTask(t, store, device, "fault")
	lifecycle := NewTransferLifecycle(db)
	at := time.Now().UTC()
	if err := lifecycle.OnUploadResponse(context.Background(), *task.CommandID, 1, at); err != nil {
		t.Fatalf("UploadResponse: %v", err)
	}
	var waiting model.TransferTask
	db.First(&waiting, "task_id = ?", task.TaskID)
	artifact := makeLifecycleArtifactAvailable(t, store, waiting, "fault", at.Add(time.Second))
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ID, at.Add(time.Second)); err != nil {
		t.Fatalf("artifact available: %v", err)
	}
	if err := lifecycle.OnTransferComplete(context.Background(), *task.CommandKey, 9010, "download failure", at.Add(3*time.Second)); err != nil {
		t.Fatalf("fault TransferComplete: %v", err)
	}
	var failed model.TransferTask
	db.First(&failed, "task_id = ?", task.TaskID)
	if failed.Status != model.TransferStatusFailed || failed.FailureCode != "9010" {
		t.Fatalf("failed task=%#v", failed)
	}
	var retained model.Artifact
	db.First(&retained, "id = ?", artifact.ID)
	if retained.Status != model.ArtifactStatusAvailable {
		t.Fatalf("artifact status=%s", retained.Status)
	}
	version := failed.Version
	if err := lifecycle.OnTransferComplete(context.Background(), *task.CommandKey, 9010, "download failure", at.Add(2*time.Second)); err != nil {
		t.Fatalf("duplicate TransferComplete: %v", err)
	}
	db.First(&failed, "task_id = ?", task.TaskID)
	if failed.Version != version {
		t.Fatalf("duplicate event changed version %d -> %d", version, failed.Version)
	}
}

func TestTransferLifecycleUnknownCommandKey(t *testing.T) {
	_, db, _ := newTransferStoreTest(t)
	err := NewTransferLifecycle(db).OnTransferComplete(context.Background(), "unknown", 0, "", time.Now())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown CommandKey error=%v", err)
	}
}
