package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeviceParameterSyncParams(t *testing.T) {
	want := map[string]interface{}{"paths": []string{"Device."}}
	if got := deviceParameterSyncParams(); !reflect.DeepEqual(got, want) {
		t.Fatalf("deviceParameterSyncParams() = %#v, want %#v", got, want)
	}
}

func TestCommandTargetByIDRequiresRecentInform(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	devices := []model.Device{
		{SerialNumber: "ONLINE", OUI: "8CE468", LastInform: now.Add(-commandOnlineThreshold + time.Second)},
		{SerialNumber: "BOUNDARY", OUI: "8CE468", LastInform: now.Add(-commandOnlineThreshold)},
		{SerialNumber: "NEVER", OUI: "8CE468"},
	}
	for i := range devices {
		if err := db.Create(&devices[i]).Error; err != nil {
			t.Fatalf("create device: %v", err)
		}
	}

	service := new(CommandService)
	key, err := service.commandTargetByID(devices[0].ID, now)
	if err != nil {
		t.Fatalf("online device rejected: %v", err)
	}
	if key != "8CE468-ONLINE" {
		t.Fatalf("device key = %q, want %q", key, "8CE468-ONLINE")
	}

	for _, device := range devices[1:] {
		_, err := service.commandTargetByID(device.ID, now)
		if !errors.Is(err, ErrDeviceOffline) {
			t.Fatalf("device %q error = %v, want ErrDeviceOffline", device.SerialNumber, err)
		}
	}
}
