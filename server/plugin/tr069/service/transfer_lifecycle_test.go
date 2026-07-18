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
		ArtifactID: "lifecycle-artifact-" + suffix, ObjectKey: "artifacts/log/" + suffix, Driver: "memory", CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("create active receiving artifact: %v", err)
	}
	_ = transitioned
	available, err := store.MarkArtifactAvailable(context.Background(), artifact.ArtifactID, artifact.Version, ArtifactFinalization{Size: 10, SHA256: "sha-" + suffix, ReceivedAt: at})
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
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ArtifactID, at.Add(time.Second)); err != nil {
		t.Fatalf("artifact available: %v", err)
	}
	var completed model.TransferTask
	db.First(&completed, "task_id = ?", task.TaskID)
	if completed.Status != model.TransferStatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("completed task=%#v", completed)
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
	if err := lifecycle.OnTransferComplete(context.Background(), *task.CommandKey, 0, "", at.Add(time.Second)); err != nil {
		t.Fatalf("TransferComplete: %v", err)
	}
	var beforeFile model.TransferTask
	db.First(&beforeFile, "task_id = ?", task.TaskID)
	if beforeFile.Status != model.TransferStatusWaitingFile || beforeFile.TransferCompletedAt == nil {
		t.Fatalf("task before file=%#v", beforeFile)
	}
	artifact := makeLifecycleArtifactAvailable(t, store, beforeFile, "status-one", at.Add(2*time.Second))
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ArtifactID, at.Add(2*time.Second)); err != nil {
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
	if err := lifecycle.OnArtifactAvailable(context.Background(), task.TaskID, artifact.ArtifactID, at.Add(time.Second)); err != nil {
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
	db.First(&retained, "artifact_id = ?", artifact.ArtifactID)
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
