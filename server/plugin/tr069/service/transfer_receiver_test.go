package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
)

type staticUploadIdentityResolver struct {
	candidates []UploadDeviceIdentity
	err        error
}

func (r staticUploadIdentityResolver) Resolve(context.Context, string) ([]UploadDeviceIdentity, error) {
	return append([]UploadDeviceIdentity(nil), r.candidates...), r.err
}

func TestTransferReceiverStreamsPeriodicArtifactAndCompletesTask(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	objects := newMemoryArtifactStore()
	receiver := NewTransferReceiver(store, objects)
	receivedAt := time.Date(2026, 7, 19, 6, 0, 0, 0, time.UTC)
	receiver.now = func() time.Time { return receivedAt }
	payload := []byte("base-station-log-archive")
	artifact, err := receiver.Receive(context.Background(), ReceiveRequest{
		Device:  UploadDeviceIdentity{DeviceID: device.ID, SerialNumber: device.SerialNumber, OUI: device.OUI},
		Channel: "LOG", Body: bytes.NewReader(payload), ContentLength: int64(len(payload)),
		OriginalName: "../bs-log.tar.gz", ContentType: "application/gzip", SourceIP: "192.0.2.10",
		Driver: "memory", StoragePrefix: "artifacts", MaxFileSize: 1024, UploadTimeout: time.Minute,
		RetentionDays: 30, MaxConcurrent: 4, MaxConcurrentPerDevice: 1,
	})
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	wantHash := sha256.Sum256(payload)
	if artifact.Status != model.ArtifactStatusAvailable || artifact.Size != int64(len(payload)) || artifact.SHA256 != hex.EncodeToString(wantHash[:]) || artifact.OriginalName != "bs-log.tar.gz" {
		t.Fatalf("artifact=%#v", artifact)
	}
	reader, _, err := objects.Open(context.Background(), artifact.ObjectKey)
	if err != nil {
		t.Fatalf("open stored artifact: %v", err)
	}
	stored, _ := io.ReadAll(reader)
	_ = reader.Close()
	if !bytes.Equal(stored, payload) {
		t.Fatalf("stored=%q", stored)
	}
	var task model.TransferTask
	if err := db.First(&task, "task_id = ?", artifact.TaskID).Error; err != nil || task.Status != model.TransferStatusCompleted {
		t.Fatalf("task=%#v err=%v", task, err)
	}
}

func TestTransferReceiverRejectsOversizedStreamAndAbortsObject(t *testing.T) {
	store, _, device := newTransferStoreTest(t)
	objects := newMemoryArtifactStore()
	receiver := NewTransferReceiver(store, objects)
	_, err := receiver.Receive(context.Background(), ReceiveRequest{
		Device:  UploadDeviceIdentity{DeviceID: device.ID, SerialNumber: device.SerialNumber, OUI: device.OUI},
		Channel: "LOG", Body: bytes.NewReader([]byte("123456")), ContentLength: -1,
		Driver: "memory", StoragePrefix: "artifacts", MaxFileSize: 5, UploadTimeout: time.Minute,
		RetentionDays: 30, MaxConcurrent: 4, MaxConcurrentPerDevice: 1,
	})
	if !errors.Is(err, ErrTransferFileTooLarge) {
		t.Fatalf("oversized stream error=%v", err)
	}
	objects.mu.Lock()
	count := len(objects.objects)
	objects.mu.Unlock()
	if count != 0 {
		t.Fatalf("aborted object count=%d", count)
	}
}

func TestTransferAdmissionRejectsGlobalAndPerDeviceSaturation(t *testing.T) {
	admission := NewTransferAdmissionController()
	release, err := admission.Acquire(1, "LOG", 2, 1)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()
	if _, err := admission.Acquire(1, "LOG", 2, 1); !errors.Is(err, ErrTransferBusy) {
		t.Fatalf("same device acquire error=%v", err)
	}
	releaseSecond, err := admission.Acquire(2, "LOG", 2, 1)
	if err != nil {
		t.Fatalf("second device acquire: %v", err)
	}
	defer releaseSecond()
	if _, err := admission.Acquire(3, "LOG", 2, 1); !errors.Is(err, ErrTransferBusy) {
		t.Fatalf("global acquire error=%v", err)
	}
}

func TestUploadDeviceResolverUsesActiveTaskThenRejectsAmbiguousFallback(t *testing.T) {
	store, db, first := newTransferStoreTest(t)
	now := time.Now().UTC()
	if err := db.Model(&first).Updates(map[string]any{"ip": "192.0.2.90", "last_inform": now, "oui": "8CE468"}).Error; err != nil {
		t.Fatalf("update first device: %v", err)
	}
	second := model.Device{SerialNumber: "BS-RESOLVE-2", OUI: "001122", IP: "192.0.2.90", LastInform: now}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second device: %v", err)
	}
	command := model.Command{CommandID: "resolve-upload", DeviceID: first.ID, Status: model.CommandStatusQueued, Operation: "Upload", QueuedAt: now, CreatedAt: now}
	commandID := command.CommandID
	task := model.TransferTask{TaskID: "resolve-active", DeviceID: first.ID, Channel: "LOG", Source: model.TransferSourceActive, Status: model.TransferStatusWaitingFile, CommandID: &commandID}
	if err := store.CreateActive(context.Background(), &command, &task); err != nil {
		t.Fatalf("create active task: %v", err)
	}
	resolver := NewUploadDeviceResolver(db, store, staticUploadIdentityResolver{}, 30*time.Minute)
	resolved, err := resolver.Resolve(context.Background(), "192.0.2.90", "LOG")
	if err != nil || resolved.DeviceID != first.ID {
		t.Fatalf("active resolution=%#v err=%v", resolved, err)
	}

	if err := db.Model(new(model.TransferTask)).Where("task_id = ?", task.TaskID).Update("status", model.TransferStatusCompleted).Error; err != nil {
		t.Fatalf("complete active task: %v", err)
	}
	_, err = resolver.Resolve(context.Background(), "192.0.2.90", "LOG")
	if !errors.Is(err, ErrUploadDeviceAmbiguous) {
		t.Fatalf("ambiguous fallback error=%v", err)
	}
}

func TestUploadDeviceResolverValidatesLiveRedisCandidateAgainstDatabase(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	now := time.Now().UTC()
	if err := db.Model(&device).Updates(map[string]any{"ip": "192.0.2.91", "last_inform": now, "oui": "8CE468"}).Error; err != nil {
		t.Fatalf("update device: %v", err)
	}
	resolver := NewUploadDeviceResolver(db, store, staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{{DeviceID: device.ID, IP: "192.0.2.91"}}}, 30*time.Minute)
	resolved, err := resolver.Resolve(context.Background(), "192.0.2.91", "LOG")
	if err != nil || resolved.DeviceID != device.ID || resolved.SerialNumber != device.SerialNumber {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}

	resolver.identities = staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{{DeviceID: 999999, IP: "192.0.2.91"}}}
	if _, err := resolver.Resolve(context.Background(), "198.51.100.200", "LOG"); !errors.Is(err, ErrUploadDeviceNotFound) && !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown device error=%v", err)
	}
}
