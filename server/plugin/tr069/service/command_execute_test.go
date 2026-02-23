package service

import (
	"context"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestWaitCommandFinal_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Command{}, &model.DataModelValue{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	prev := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = prev })

	cmdID := "cmd-1"
	if err := db.Create(&model.Command{CommandID: cmdID, Status: "PENDING", CreatedAt: time.Now(), UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("create cmd: %v", err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = db.Model(&model.Command{}).Where("command_id = ?", cmdID).Update("status", "SUCCESS").Error
	}()

	status, err := waitCommandFinal(context.Background(), cmdID, 2*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if status != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", status)
	}
}

func TestReadDataModelValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Command{}, &model.DataModelValue{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	prev := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = prev })

	if err := db.Create(&model.DataModelValue{
		DeviceID:  1,
		Name:      "Device.DeviceInfo.SerialNumber",
		ValueJSON: []byte(`"SN123"`),
	}).Error; err != nil {
		t.Fatalf("create dm: %v", err)
	}

	out, err := readDataModelValues(1, []string{"Device.DeviceInfo.SerialNumber"})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out["Device.DeviceInfo.SerialNumber"] != "SN123" {
		t.Fatalf("expected SN123, got %#v", out["Device.DeviceInfo.SerialNumber"])
	}
}
