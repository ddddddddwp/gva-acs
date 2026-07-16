package initialize

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestSanitizeStoredCommandXMLTargetsMatchingRowsAndDeletesMalformed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.CommandXML)); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	records := []model.CommandXML{
		{CommandID: "valid-match", Payload: []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>historical-secret</Value></ParameterValueStruct></Envelope>`)},
		{CommandID: "malformed-match", Payload: []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>malformed-secret</Envelope>`)},
		{CommandID: "unrelated", Payload: []byte(`<Download><Password>download-secret</Password></Download>`)},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed XML: %v", err)
	}

	if err := sanitizeStoredCommandXML(context.Background(), db); err != nil {
		t.Fatalf("sanitize stored XML: %v", err)
	}

	var valid model.CommandXML
	if err := db.First(&valid, "command_id = ?", "valid-match").Error; err != nil {
		t.Fatalf("load sanitized XML: %v", err)
	}
	if strings.Contains(string(valid.Payload), "historical-secret") || !strings.Contains(string(valid.Payload), "******") {
		t.Fatalf("matching XML was not sanitized: %s", valid.Payload)
	}
	var malformedCount int64
	if err := db.Model(new(model.CommandXML)).Where("command_id = ?", "malformed-match").Count(&malformedCount).Error; err != nil {
		t.Fatal(err)
	}
	if malformedCount != 0 {
		t.Fatalf("malformed matching rows = %d, want 0", malformedCount)
	}
	var unrelated model.CommandXML
	if err := db.First(&unrelated, "command_id = ?", "unrelated").Error; err != nil {
		t.Fatalf("load unrelated XML: %v", err)
	}
	if string(unrelated.Payload) != `<Download><Password>download-secret</Password></Download>` {
		t.Fatalf("unrelated XML changed: %s", unrelated.Payload)
	}
	var markerCount int64
	if err := db.Model(new(tr069MigrationMarker)).Where("name = ?", commandXMLRedactionMigration).Count(&markerCount).Error; err != nil {
		t.Fatalf("count migration markers: %v", err)
	}
	if markerCount != 1 {
		t.Fatalf("migration markers = %d, want 1", markerCount)
	}

	late := model.CommandXML{CommandID: "after-migration", Payload: []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>late-secret</Value></ParameterValueStruct></Envelope>`)}
	if err := db.Create(&late).Error; err != nil {
		t.Fatalf("seed post-migration XML: %v", err)
	}
	if err := sanitizeStoredCommandXML(context.Background(), db); err != nil {
		t.Fatalf("repeat sanitize stored XML: %v", err)
	}
	var gotLate model.CommandXML
	if err := db.First(&gotLate, late.ID).Error; err != nil {
		t.Fatalf("load post-migration XML: %v", err)
	}
	if !strings.Contains(string(gotLate.Payload), "late-secret") {
		t.Fatalf("completed migration reran unexpectedly: %s", gotLate.Payload)
	}
}

func TestSanitizeStoredCommandXMLUsesCandidateQueryAndBoundedBatches(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.CommandXML), new(tr069MigrationMarker)); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register("test:bounded_command_xml_candidates", func(tx *gorm.DB) {
		if tx.Statement.Schema == nil || tx.Statement.Schema.Name != "CommandXML" {
			return
		}
		if _, ok := tx.Statement.Clauses["WHERE"]; !ok {
			tx.AddError(errors.New("command XML candidate query has no WHERE clause"))
			return
		}
		if _, ok := tx.Statement.Clauses["LIMIT"]; !ok {
			tx.AddError(errors.New("command XML candidate query has no LIMIT clause"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	records := make([]model.CommandXML, 0, 101)
	for index := 0; index < 101; index++ {
		records = append(records, model.CommandXML{CommandID: fmt.Sprintf("batch-%03d", index), Payload: []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>batch-secret</Value></ParameterValueStruct></Envelope>`)})
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatal(err)
	}

	if err := sanitizeStoredCommandXML(context.Background(), db); err != nil {
		t.Fatalf("sanitize stored XML: %v", err)
	}
	if err := db.Callback().Query().Remove("test:bounded_command_xml_candidates"); err != nil {
		t.Fatal(err)
	}
	var stored []model.CommandXML
	if err := db.Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	for _, record := range stored {
		if strings.Contains(string(record.Payload), "batch-secret") {
			t.Fatalf("candidate %d was not sanitized", record.ID)
		}
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
