package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
)

type failingDeleteArtifactStore struct {
	ArtifactStore
	err error
}

func (s failingDeleteArtifactStore) Delete(context.Context, string) error { return s.err }

func configureTransferWorkerRuntime(t *testing.T, uploadTimeout time.Duration) {
	t.Helper()
	previous := config.CurrentRuntime()
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })
	config.StoreRuntime(config.TR069Config{FileIngress: config.FileIngressConfig{
		Enabled: true,
		Channels: map[string]config.TransferChannelConfig{"log": {
			Enabled: true, Path: "/acs/log", UploadTimeout: int(uploadTimeout.Seconds()), RetentionDays: 30,
		}},
	}})
}

func TestTransferWorkerReconcilesCommittedStaleReceivingObject(t *testing.T) {
	configureTransferWorkerRuntime(t, 10*time.Second)
	store, db, device := newTransferStoreTest(t)
	objects := newMemoryArtifactStore()
	now := time.Date(2026, 7, 19, 9, 0, 0, 0, time.UTC)
	task, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{
		TaskID: "reconcile-task", StoragePrefix: "artifacts", Driver: "memory", CreatedAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("create receiving metadata: %v", err)
	}
	w, _ := objects.Begin(context.Background(), ObjectSpec{Key: artifact.ObjectKey})
	_, _ = w.Write([]byte("recovered-log"))
	if _, err := w.Commit(context.Background()); err != nil {
		t.Fatalf("commit recovered object: %v", err)
	}
	workers := NewTransferWorkers(store, objects)
	workers.now = func() time.Time { return now }
	if err := workers.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	var recovered model.Artifact
	db.First(&recovered, "id = ?", artifact.ID)
	if recovered.Status != model.ArtifactStatusAvailable || recovered.Size != int64(len("recovered-log")) || recovered.SHA256 == "" {
		t.Fatalf("recovered artifact=%#v", recovered)
	}
	var completed model.TransferTask
	db.First(&completed, "task_id = ?", task.TaskID)
	if completed.Status != model.TransferStatusCompleted {
		t.Fatalf("task status=%s", completed.Status)
	}
}

func TestTransferWorkerFailsStaleReceivingWithoutObjectAndTimesOutTask(t *testing.T) {
	configureTransferWorkerRuntime(t, 10*time.Second)
	store, db, device := newTransferStoreTest(t)
	objects := newMemoryArtifactStore()
	now := time.Date(2026, 7, 19, 9, 5, 0, 0, time.UTC)
	task, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{
		TaskID: "missing-task", StoragePrefix: "artifacts", Driver: "memory", CreatedAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("create missing metadata: %v", err)
	}
	deadline := now.Add(-time.Second)
	if err := db.Model(new(model.TransferTask)).Where("task_id = ?", task.TaskID).Update("phase_deadline_at", deadline).Error; err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	workers := NewTransferWorkers(store, objects)
	workers.now = func() time.Time { return now }
	if err := workers.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	var failedArtifact model.Artifact
	db.First(&failedArtifact, "id = ?", artifact.ID)
	if failedArtifact.Status != model.ArtifactStatusFailed {
		t.Fatalf("artifact status=%s", failedArtifact.Status)
	}
	var terminal model.TransferTask
	db.First(&terminal, "task_id = ?", task.TaskID)
	if terminal.Status != model.TransferStatusFailed && terminal.Status != model.TransferStatusTimeout {
		t.Fatalf("task status=%s", terminal.Status)
	}
}

func TestTransferWorkerRetentionDeletesAndRetriesFailures(t *testing.T) {
	configureTransferWorkerRuntime(t, 10*time.Second)
	store, db, device := newTransferStoreTest(t)
	objects := newMemoryArtifactStore()
	now := time.Date(2026, 7, 19, 9, 10, 0, 0, time.UTC)
	task, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{
		TaskID: "retention-task", StoragePrefix: "artifacts", Driver: "memory", CreatedAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("create retention metadata: %v", err)
	}
	w, _ := objects.Begin(context.Background(), ObjectSpec{Key: artifact.ObjectKey})
	_, _ = w.Write([]byte("retained"))
	_, _ = w.Commit(context.Background())
	available, err := store.MarkArtifactAvailable(context.Background(), artifact.ID, artifact.Version, ArtifactFinalization{Size: 8, SHA256: "sha", ReceivedAt: now.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("mark available: %v", err)
	}
	if err := NewTransferLifecycle(db).OnArtifactAvailable(context.Background(), task.TaskID, artifact.ID, now.Add(-time.Hour)); err != nil {
		t.Fatalf("record available: %v", err)
	}
	if err := db.Model(new(model.Artifact)).Where("id = ?", artifact.ID).Update("delete_at", now.Add(-time.Second)).Error; err != nil {
		t.Fatalf("set retention deadline: %v", err)
	}
	injected := errors.New("delete unavailable")
	failing := NewTransferWorkers(store, failingDeleteArtifactStore{ArtifactStore: objects, err: injected})
	failing.now = func() time.Time { return now }
	if err := failing.RunOnce(context.Background()); err == nil {
		t.Fatal("delete failure was not reported")
	}
	var deleting model.Artifact
	db.First(&deleting, "id = ?", available.ID)
	if deleting.Status != model.ArtifactStatusDeleting {
		t.Fatalf("status after failed delete=%s", deleting.Status)
	}

	workers := NewTransferWorkers(store, objects)
	workers.now = func() time.Time { return now.Add(time.Second) }
	if err := workers.RunOnce(context.Background()); err != nil {
		t.Fatalf("retry retention: %v", err)
	}
	var deleted model.Artifact
	db.First(&deleted, "id = ?", artifact.ID)
	if deleted.Status != model.ArtifactStatusDeleted || deleted.DeletedAt == nil {
		t.Fatalf("deleted artifact=%#v", deleted)
	}
}
