package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type deletionFixture struct {
	device    model.Device
	commandID string
	taskID    string
	fileID    uint64
	objectKey string
}

type deletionArtifactStore struct {
	mu       sync.Mutex
	objects  map[string][]byte
	failKey  string
	failOnce bool
}

func (s *deletionArtifactStore) Begin(context.Context, ObjectSpec) (ArtifactWriter, error) {
	return nil, errors.New("not implemented")
}
func (s *deletionArtifactStore) Open(context.Context, string) (io.ReadCloser, ObjectStat, error) {
	return nil, ObjectStat{}, errors.New("not implemented")
}
func (s *deletionArtifactStore) Stat(context.Context, string) (ObjectStat, error) {
	return ObjectStat{}, errors.New("not implemented")
}
func (s *deletionArtifactStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == s.failKey && s.failOnce {
		s.failOnce = false
		return errors.New("storage unavailable")
	}
	if _, ok := s.objects[key]; !ok {
		return ErrArtifactNotFound
	}
	delete(s.objects, key)
	return nil
}
func (s *deletionArtifactStore) has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok
}

type deletionRuntimeCleaner struct {
	mu         sync.Mutex
	identities []DeviceRuntimeIdentity
	err        error
}

func (c *deletionRuntimeCleaner) Purge(_ context.Context, identity DeviceRuntimeIdentity) error {
	c.mu.Lock()
	c.identities = append(c.identities, identity)
	c.mu.Unlock()
	return c.err
}

func TestDeviceDeletionRemovesTargetAndPreservesPeer(t *testing.T) {
	db := newDeviceDeletionDB(t)
	target := seedDeviceDeletionFixture(t, db, "001122", "DELETE-TARGET")
	peer := seedDeviceDeletionFixture(t, db, "334455", "DELETE-PEER")
	objects := &deletionArtifactStore{objects: map[string][]byte{
		target.objectKey: []byte("target-log"),
		peer.objectKey:   []byte("peer-log"),
	}}
	runtime := new(deletionRuntimeCleaner)
	uploads := NewUploadRuntimeRegistry()
	service := NewDeviceDeletionService(db, objects, uploads, runtime)

	result, err := service.Delete(context.Background(), target.device.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if result.DeviceID != target.device.ID || result.DeletedObjects != 1 {
		t.Fatalf("Delete() result = %#v", result)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 0)
	assertDeviceDeletionFixtureCount(t, db, peer, 1)
	if objects.has(target.objectKey) {
		t.Fatal("target object still exists")
	}
	if !objects.has(peer.objectKey) {
		t.Fatal("peer object was deleted")
	}
	if len(runtime.identities) != 1 || runtime.identities[0].DeviceID != target.device.ID || runtime.identities[0].DeviceKey != "001122-DELETE-TARGET" {
		t.Fatalf("runtime identities = %#v", runtime.identities)
	}
	if handle, err := uploads.Register(target.device.ID, func() {}); err != nil {
		t.Fatalf("successful deletion did not release upload block: %v", err)
	} else {
		handle.Unregister()
	}
}

func TestDeviceDeletionStorageFailureKeepsDatabaseAndDeletingMarker(t *testing.T) {
	db := newDeviceDeletionDB(t)
	target := seedDeviceDeletionFixture(t, db, "001122", "DELETE-STORAGE-FAIL")
	objects := &deletionArtifactStore{
		objects: map[string][]byte{target.objectKey: []byte("target-log")},
		failKey: target.objectKey, failOnce: true,
	}
	uploads := NewUploadRuntimeRegistry()
	service := NewDeviceDeletionService(db, objects, uploads, new(deletionRuntimeCleaner))

	_, err := service.Delete(context.Background(), target.device.ID)
	var deletionErr *DeviceDeletionError
	if !errors.As(err, &deletionErr) || deletionErr.Stage != DeletionStageArtifacts {
		t.Fatalf("Delete() error = %v, want artifact stage", err)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 1)
	var kept model.Device
	if err := db.First(&kept, target.device.ID).Error; err != nil || kept.DeletingAt == nil {
		t.Fatalf("kept device = %#v, error = %v", kept, err)
	}
	if _, err := uploads.Register(target.device.ID, func() {}); !errors.Is(err, ErrDeviceDeleting) {
		t.Fatalf("upload block error = %v, want ErrDeviceDeleting", err)
	}

	if _, err := service.Delete(context.Background(), target.device.ID); err != nil {
		t.Fatalf("retry Delete() error = %v", err)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 0)
}

func TestDeviceDeletionRetryTreatsMissingObjectsAsSuccess(t *testing.T) {
	db := newDeviceDeletionDB(t)
	target := seedDeviceDeletionFixture(t, db, "001122", "DELETE-MISSING-OBJECT")
	objects := &deletionArtifactStore{objects: map[string][]byte{}}
	service := NewDeviceDeletionService(db, objects, NewUploadRuntimeRegistry(), new(deletionRuntimeCleaner))

	if _, err := service.Delete(context.Background(), target.device.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 0)
}

func TestDeviceDeletionDatabaseFailureKeepsAllDatabaseRows(t *testing.T) {
	db := newDeviceDeletionDB(t)
	target := seedDeviceDeletionFixture(t, db, "001122", "DELETE-DB-FAIL")
	objects := &deletionArtifactStore{objects: map[string][]byte{target.objectKey: []byte("target-log")}}
	service := NewDeviceDeletionService(db, objects, NewUploadRuntimeRegistry(), new(deletionRuntimeCleaner))
	if err := db.Exec(`CREATE TRIGGER prevent_device_delete BEFORE DELETE ON tr069_devices BEGIN SELECT RAISE(FAIL, 'blocked delete'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	_, err := service.Delete(context.Background(), target.device.ID)
	var deletionErr *DeviceDeletionError
	if !errors.As(err, &deletionErr) || deletionErr.Stage != DeletionStageDatabase {
		t.Fatalf("Delete() error = %v, want database stage", err)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 1)
	if err := db.Exec(`DROP TRIGGER prevent_device_delete`).Error; err != nil {
		t.Fatalf("drop failure trigger: %v", err)
	}
	if _, err := service.Delete(context.Background(), target.device.ID); err != nil {
		t.Fatalf("retry Delete() error = %v", err)
	}
	assertDeviceDeletionFixtureCount(t, db, target, 0)
}

func TestDeviceDeletionMissingDeviceReturnsNotFound(t *testing.T) {
	db := newDeviceDeletionDB(t)
	service := NewDeviceDeletionService(db, &deletionArtifactStore{objects: map[string][]byte{}}, NewUploadRuntimeRegistry(), new(deletionRuntimeCleaner))
	if _, err := service.Delete(context.Background(), 999999); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("Delete() error = %v, want ErrDeviceNotFound", err)
	}
}

func newDeviceDeletionDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.Command), new(model.CommandEvent), new(model.CommandXML),
		new(model.DataModelValue), new(model.DeviceRPCMethods), new(model.FAPService),
		new(model.Tr069Alarm), new(model.SupportTr069Alarm), new(model.ConnectionProfile), new(model.TransferTask),
		new(model.Artifact), new(model.TransferEvent),
	); err != nil {
		t.Fatalf("migrate deletion models: %v", err)
	}
	return db
}

func seedDeviceDeletionFixture(t *testing.T, db *gorm.DB, oui, serial string) deletionFixture {
	t.Helper()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	device := model.Device{OUI: oui, SerialNumber: serial, ProductClass: "NR-BS", IP: "192.0.2." + fmt.Sprint(len(serial)), LastInform: now}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	commandID := "command-" + serial
	taskID := "task-" + serial
	command := model.Command{CommandID: commandID, DeviceID: device.ID, DeviceKey: oui + "-" + serial, Operation: "Upload", Status: model.CommandStatusCompleted, QueuedAt: now, CreatedAt: now, UpdatedAt: now}
	rows := []any{
		&model.DataModelValue{DeviceID: device.ID, Name: "Device.DeviceInfo.SerialNumber", ValueJSON: []byte(`"` + serial + `"`), LastCollectedAt: now},
		&model.DeviceRPCMethods{DeviceID: device.ID, MethodsJSON: datatypes.JSON(`["Upload"]`)},
		&model.FAPService{DeviceID: device.ID, CellID: "cell-" + serial},
		&model.Tr069Alarm{DeviceID: device.ID, SerialNumber: serial, AlarmIdentifier: "alarm-" + serial, Status: "Active", EventTime: now, StartTime: now, LastChanged: now},
		&model.SupportTr069Alarm{DeviceID: device.ID, SerialNumber: serial, EventType: "support-alarm", PerceivedSeverity: "Major"},
		&model.ConnectionProfile{DeviceID: device.ID, DiscoveredURL: "http://" + device.IP + ":7547", CreatedAt: now, UpdatedAt: now},
		&command,
		&model.CommandEvent{CommandID: commandID, EventType: "CREATED", ToStatus: model.CommandStatusCompleted, CreatedAt: now},
		&model.CommandXML{CommandID: commandID, Direction: "OUTBOUND", Method: "Upload", CWMPID: "cwmp-" + serial, Payload: []byte("<xml/>"), ExpiresAt: now.Add(time.Hour), CreatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create %T: %v", row, err)
		}
	}
	commandIDValue := commandID
	task := model.TransferTask{TaskID: taskID, DeviceID: device.ID, Channel: "LOG", Source: model.TransferSourceActive, CommandID: &commandIDValue, Status: model.TransferStatusCompleted, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create transfer task: %v", err)
	}
	artifact := model.Artifact{TaskID: taskID, DeviceID: device.ID, Channel: "LOG", Status: model.ArtifactStatusAvailable, Driver: "minio", ObjectKey: "tr069/log/" + serial, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&artifact).Error; err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	if err := db.Create(&model.TransferEvent{TaskID: taskID, FileID: artifact.ID, Code: "COMPLETED", CreatedAt: now}).Error; err != nil {
		t.Fatalf("create transfer event: %v", err)
	}
	return deletionFixture{device: device, commandID: commandID, taskID: taskID, fileID: artifact.ID, objectKey: artifact.ObjectKey}
}

func assertDeviceDeletionFixtureCount(t *testing.T, db *gorm.DB, fixture deletionFixture, want int64) {
	t.Helper()
	checks := []struct {
		name  string
		model any
		query string
		value any
	}{
		{"device", new(model.Device), "id = ?", fixture.device.ID},
		{"value", new(model.DataModelValue), "device_id = ?", fixture.device.ID},
		{"rpc methods", new(model.DeviceRPCMethods), "device_id = ?", fixture.device.ID},
		{"fap", new(model.FAPService), "device_id = ?", fixture.device.ID},
		{"alarm", new(model.Tr069Alarm), "device_id = ?", fixture.device.ID},
		{"support alarm", new(model.SupportTr069Alarm), "device_id = ?", fixture.device.ID},
		{"profile", new(model.ConnectionProfile), "device_id = ?", fixture.device.ID},
		{"command", new(model.Command), "device_id = ?", fixture.device.ID},
		{"command event", new(model.CommandEvent), "command_id = ?", fixture.commandID},
		{"command xml", new(model.CommandXML), "command_id = ?", fixture.commandID},
		{"transfer task", new(model.TransferTask), "device_id = ?", fixture.device.ID},
		{"artifact", new(model.Artifact), "device_id = ?", fixture.device.ID},
		{"transfer event", new(model.TransferEvent), "task_id = ?", fixture.taskID},
	}
	for _, check := range checks {
		var count int64
		if err := db.Unscoped().Model(check.model).Where(check.query, check.value).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", check.name, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", check.name, count, want)
		}
	}
}
