package initialize

import (
	"context"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestConnectionProfileMigrationBackfillsDeviceURL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	device := model.Device{SerialNumber: "PROFILE-BACKFILL", ConnectionReqURL: "http://127.0.0.1:8400"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	if err := migrateConnectionProfiles(context.Background(), db); err != nil {
		t.Fatalf("migrate connection profiles: %v", err)
	}
	if err := migrateConnectionProfiles(context.Background(), db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}

	var profiles []model.ConnectionProfile
	if err := db.Where("device_id = ?", device.ID).Find(&profiles).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles=%d, want exactly one", len(profiles))
	}
	if profiles[0].DiscoveredURL != device.ConnectionReqURL {
		t.Fatalf("discovered URL=%q, want %q", profiles[0].DiscoveredURL, device.ConnectionReqURL)
	}
	if profiles[0].ProvisionState != model.ConnectionProfileStateDiscovered {
		t.Fatalf("provision state=%q", profiles[0].ProvisionState)
	}
}

func TestConnectionProfileMigrationIgnoresDevicesWithoutURL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	device := model.Device{SerialNumber: "PROFILE-NO-URL"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	if err := migrateConnectionProfiles(context.Background(), db); err != nil {
		t.Fatalf("migrate connection profiles: %v", err)
	}

	var count int64
	if err := db.Model(new(model.ConnectionProfile)).Where("device_id = ?", device.ID).Count(&count).Error; err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	if count != 0 {
		t.Fatalf("profiles=%d, want 0", count)
	}
}
