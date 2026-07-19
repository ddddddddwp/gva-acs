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

func TestTransferReceiverMakesActiveUploadRetriesIdempotentByContent(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	now := time.Now().UTC()
	commandID := "upload-retry-command"
	commandKey := "uploadretrycommand000000000000001"
	command := model.Command{
		CommandID: commandID, DeviceID: device.ID, DeviceKey: device.SerialNumber,
		Operation: "Upload", Status: model.CommandStatusSent, CommandKey: &commandKey,
		QueuedAt: now, CreatedAt: now,
	}
	task := model.TransferTask{
		TaskID: "upload-retry-task", DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive,
		CommandID: &commandID, CommandKey: &commandKey, Status: model.TransferStatusWaitingFile, CreatedAt: now,
	}
	if err := store.CreateActive(context.Background(), &command, &task); err != nil {
		t.Fatalf("create active task: %v", err)
	}
	objects := newMemoryArtifactStore()
	receiver := NewTransferReceiver(store, objects)
	payload := []byte("same-active-log")
	request := func(body []byte) ReceiveRequest {
		return ReceiveRequest{
			Device:  UploadDeviceIdentity{DeviceID: device.ID, SerialNumber: device.SerialNumber, OUI: device.OUI},
			Channel: "LOG", Body: bytes.NewReader(body), ContentLength: int64(len(body)),
			Driver: "memory", StoragePrefix: "artifacts", MaxFileSize: 1024, UploadTimeout: time.Minute,
			RetentionDays: 30, MaxConcurrent: 4, MaxConcurrentPerDevice: 1,
		}
	}
	first, err := receiver.Receive(context.Background(), request(payload))
	if err != nil {
		t.Fatalf("first receive: %v", err)
	}
	duplicate, err := receiver.Receive(context.Background(), request(payload))
	if err != nil || duplicate.ID != first.ID {
		t.Fatalf("same-content retry artifact=%#v err=%v", duplicate, err)
	}
	if _, err := receiver.Receive(context.Background(), request([]byte("different-active-log"))); !errors.Is(err, ErrTransferContentConflict) {
		t.Fatalf("different-content retry error=%v", err)
	}
	var artifacts, tasks int64
	if err := db.Model(new(model.Artifact)).Count(&artifacts).Error; err != nil {
		t.Fatalf("count artifacts: %v", err)
	}
	if err := db.Model(new(model.TransferTask)).Count(&tasks).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	objects.mu.Lock()
	objectsCount := len(objects.objects)
	objects.mu.Unlock()
	if artifacts != 1 || tasks != 1 || objectsCount != 1 {
		t.Fatalf("retry created extra state: artifacts=%d tasks=%d objects=%d", artifacts, tasks, objectsCount)
	}
	var retryEvents int64
	if err := db.Model(new(model.TransferEvent)).Where("task_id = ? AND code IN ?", task.TaskID, []string{"DUPLICATE_ACCEPTED", "DUPLICATE_CONTENT_CONFLICT"}).Count(&retryEvents).Error; err != nil {
		t.Fatalf("count retry events: %v", err)
	}
	if retryEvents != 2 {
		t.Fatalf("retry audit events=%d, want 2", retryEvents)
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
	if err := db.Model(&device).Updates(map[string]any{"ip": "192.0.2.91", "last_inform": now, "oui": "8CE468", "product_class": "NR-BS"}).Error; err != nil {
		t.Fatalf("update device: %v", err)
	}
	device.OUI = "8CE468"
	device.ProductClass = "NR-BS"
	exact := UploadDeviceIdentity{DeviceID: device.ID, IP: "192.0.2.91", OUI: device.OUI, ProductClass: device.ProductClass, SerialNumber: device.SerialNumber}
	resolver := NewUploadDeviceResolver(db, store, staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{exact}}, 30*time.Minute)
	resolved, err := resolver.Resolve(context.Background(), "192.0.2.91", "LOG")
	if err != nil || resolved.DeviceID != device.ID || resolved.SerialNumber != device.SerialNumber {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
	for name, mutate := range map[string]func(*UploadDeviceIdentity){
		"missing identity": func(candidate *UploadDeviceIdentity) { candidate.OUI = "" },
		"OUI mismatch":     func(candidate *UploadDeviceIdentity) { candidate.OUI = "FFFFFF" },
		"product mismatch": func(candidate *UploadDeviceIdentity) { candidate.ProductClass = "OTHER" },
		"serial mismatch":  func(candidate *UploadDeviceIdentity) { candidate.SerialNumber = "OTHER-SN" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := exact
			mutate(&candidate)
			resolver.identities = staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{candidate}}
			if _, err := resolver.Resolve(context.Background(), "192.0.2.91", "LOG"); !errors.Is(err, ErrUploadDeviceNotFound) {
				t.Fatalf("identity conflict error=%v", err)
			}
		})
	}

	resolver.identities = staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{{DeviceID: 999999, IP: "192.0.2.91"}}}
	if _, err := resolver.Resolve(context.Background(), "198.51.100.200", "LOG"); !errors.Is(err, ErrUploadDeviceNotFound) && !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown device error=%v", err)
	}
}

func TestUploadDeviceResolverRejectsDeletingDevice(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	now := time.Now().UTC()
	if err := db.Model(&device).Updates(map[string]any{
		"ip": "192.0.2.92", "last_inform": now, "product_class": "NR-BS", "deleting_at": now,
	}).Error; err != nil {
		t.Fatalf("mark deleting device: %v", err)
	}
	identity := UploadDeviceIdentity{
		DeviceID: device.ID, IP: "192.0.2.92", OUI: device.OUI,
		ProductClass: "NR-BS", SerialNumber: device.SerialNumber,
	}
	resolver := NewUploadDeviceResolver(db, store, staticUploadIdentityResolver{candidates: []UploadDeviceIdentity{identity}}, 30*time.Minute)
	if _, err := resolver.Resolve(context.Background(), identity.IP, "LOG"); !errors.Is(err, ErrUploadDeviceNotFound) {
		t.Fatalf("Resolve() error = %v, want ErrUploadDeviceNotFound", err)
	}
}
