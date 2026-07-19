package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestArtifactUsesAutoIncrementFileIDWithoutUUID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Artifact), new(model.TransferEvent)); err != nil {
		t.Fatalf("migrate artifacts: %v", err)
	}

	first := model.Artifact{TaskID: "file-id-task-1", DeviceID: 1, Channel: "LOG", Status: model.ArtifactStatusReceiving, ObjectKey: "log/1"}
	second := model.Artifact{TaskID: "file-id-task-2", DeviceID: 1, Channel: "LOG", Status: model.ArtifactStatusReceiving, ObjectKey: "log/2"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first artifact: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second artifact: %v", err)
	}
	if first.ID == 0 || second.ID != first.ID+1 {
		t.Fatalf("file IDs = %d, %d", first.ID, second.ID)
	}

	event := model.TransferEvent{TaskID: first.TaskID, FileID: first.ID, Code: "RECEIVING_STARTED", CreatedAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create transfer event: %v", err)
	}
	var stored model.TransferEvent
	if err := db.First(&stored, event.ID).Error; err != nil || stored.FileID != first.ID {
		t.Fatalf("stored event = %#v, err=%v", stored, err)
	}

	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if strings.Contains(string(encoded), "artifactId") || !strings.Contains(string(encoded), `"fileId":`) {
		t.Fatalf("artifact JSON = %s", encoded)
	}
}

func TestArtifactObjectKeyUsesNumericFileID(t *testing.T) {
	key, err := ArtifactObjectKey("artifacts", "LOG", 42, time.Date(2026, 7, 19, 4, 5, 6, 0, time.UTC), 101)
	if err != nil {
		t.Fatalf("object key: %v", err)
	}
	if key != "artifacts/log/42/2026/07/19/101" {
		t.Fatalf("object key = %q", key)
	}
}
