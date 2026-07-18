package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTransferStoreTest(t *testing.T) (*TransferStore, *gorm.DB, model.Device) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.CommandEvent), new(model.TransferTask), new(model.Artifact), new(model.TransferEvent)); err != nil {
		t.Fatalf("migrate transfer models: %v", err)
	}
	device := model.Device{SerialNumber: "BS-" + strings.ReplaceAll(t.Name(), "/", "-"), OUI: "8CE468", LastInform: time.Now().UTC()}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	return NewTransferStore(db), db, device
}

func TestTransferStoreCreatesPeriodicReceiveAtomically(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{
		TaskID:     "periodic-task-1",
		ArtifactID: "artifact-1",
		ObjectKey:  "artifacts/log/1/2026/07/19/artifact-1",
		Driver:     "minio",
		SourceIP:   "192.0.2.10",
	})
	if err != nil || task.Source != model.TransferSourcePeriodic || artifact.Status != model.ArtifactStatusReceiving {
		t.Fatalf("task=%#v artifact=%#v err=%v", task, artifact, err)
	}
	var events int64
	if err := db.Model(new(model.TransferEvent)).Where("task_id = ?", task.TaskID).Count(&events).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if events != 1 {
		t.Fatalf("events=%d", events)
	}
}

func TestTransferStoreCreateActiveRollsBackCommandWhenTaskInsertFails(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	if err := db.Create(&model.TransferTask{TaskID: "duplicate-task", DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourcePeriodic, Status: model.TransferStatusReceiving}).Error; err != nil {
		t.Fatalf("seed duplicate task: %v", err)
	}
	command := model.Command{CommandID: "command-rollback", DeviceID: device.ID, Status: model.CommandStatusQueued, Operation: "Upload", QueuedAt: time.Now(), CreatedAt: time.Now()}
	task := model.TransferTask{TaskID: "duplicate-task", DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive, Status: model.TransferStatusWaitingFile, CommandID: &command.CommandID}

	if err := store.CreateActive(context.Background(), &command, &task); err == nil {
		t.Fatal("duplicate transfer task unexpectedly succeeded")
	}
	var count int64
	db.Model(new(model.Command)).Where("command_id = ?", command.CommandID).Count(&count)
	if count != 0 {
		t.Fatalf("command survived rolled-back transaction: count=%d", count)
	}
}

func TestTransferStoreEnforcesOneActiveTaskPerCommand(t *testing.T) {
	store, _, device := newTransferStoreTest(t)
	command := model.Command{CommandID: "command-one-task", DeviceID: device.ID, Status: model.CommandStatusQueued, Operation: "Upload", QueuedAt: time.Now(), CreatedAt: time.Now()}
	commandID := command.CommandID
	first := model.TransferTask{TaskID: "active-task-1", DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive, Status: model.TransferStatusWaitingFile, CommandID: &commandID}
	if err := store.CreateActive(context.Background(), &command, &first); err != nil {
		t.Fatalf("create first active task: %v", err)
	}
	second := model.TransferTask{TaskID: "active-task-2", DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive, Status: model.TransferStatusWaitingFile, CommandID: &commandID}
	if err := store.db.Create(&second).Error; err == nil {
		t.Fatal("second active task for command unexpectedly succeeded")
	}
}

func TestTransferStoreTransitionUsesStatusAndVersion(t *testing.T) {
	store, _, device := newTransferStoreTest(t)
	task, _, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{TaskID: "transition-task", ArtifactID: "transition-artifact", ObjectKey: "log/transition", Driver: "minio"})
	if err != nil {
		t.Fatalf("create receiving task: %v", err)
	}
	transitioned, err := store.TransitionTask(context.Background(), TransferTransition{
		TaskID: task.TaskID, FromStatuses: []string{model.TransferStatusReceiving}, ToStatus: model.TransferStatusCompleted,
		ExpectedVersion: task.Version, EventCode: "FILE_STORED", Phase: "storage.commit",
	})
	if err != nil || transitioned.Status != model.TransferStatusCompleted || transitioned.Version != task.Version+1 {
		t.Fatalf("transitioned=%#v err=%v", transitioned, err)
	}
	_, err = store.TransitionTask(context.Background(), TransferTransition{
		TaskID: task.TaskID, FromStatuses: []string{model.TransferStatusReceiving}, ToStatus: model.TransferStatusFailed,
		ExpectedVersion: task.Version, EventCode: "LATE_FAILURE",
	})
	if !errors.Is(err, ErrTransferTransitionConflict) {
		t.Fatalf("stale transition error=%v", err)
	}
}

func TestTransferStoreListsArtifactsByExactDeviceIDAndHidesObjectKey(t *testing.T) {
	store, db, first := newTransferStoreTest(t)
	second := model.Device{SerialNumber: "BS-SECOND", OUI: "001122", LastInform: time.Now().UTC()}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second device: %v", err)
	}
	for index, device := range []model.Device{first, second, first} {
		taskID := fmt.Sprintf("list-task-%d", index)
		artifactID := fmt.Sprintf("list-artifact-%d", index)
		_, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{TaskID: taskID, ArtifactID: artifactID, ObjectKey: "secret/" + artifactID, Driver: "minio"})
		if err != nil {
			t.Fatalf("create artifact %d: %v", index, err)
		}
		if _, err := store.MarkArtifactAvailable(context.Background(), artifact.ArtifactID, artifact.Version, ArtifactFinalization{Size: int64(index + 1), SHA256: fmt.Sprintf("sha-%d", index), ReceivedAt: time.Now().UTC()}); err != nil {
			t.Fatalf("finalize artifact %d: %v", index, err)
		}
	}

	items, total, err := store.ListArtifacts(context.Background(), ArtifactListFilter{DeviceID: first.ID, Limit: 1, Offset: 1})
	if err != nil || total != 2 || len(items) != 1 || items[0].DeviceID != first.ID || items[0].SerialNumber != first.SerialNumber {
		t.Fatalf("items=%#v total=%d err=%v", items, total, err)
	}
	raw, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if strings.Contains(string(raw), "objectKey") || strings.Contains(string(raw), "secret/") || strings.Contains(string(raw), "password") {
		t.Fatalf("private storage data leaked in JSON: %s", raw)
	}
}
