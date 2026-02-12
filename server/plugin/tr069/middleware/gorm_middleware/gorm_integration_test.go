package gormmiddleware

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type dmValueLite struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func (dmValueLite) TableName() string {
	return "tr069_datamodel_values"
}

func TestMiddleware_GormCreateFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&dmValueLite{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rules := RulesForPrefixDeny("tr069_datamodel_values", "Name", []string{"Device.FaultMgmt."})

	denied := dmValueLite{Name: "Device.FaultMgmt.A"}
	if !DenyByPrefixes(denied.Name, rules[0].DenyPrefixes) {
		if err := db.Create(&denied).Error; err != nil {
			t.Fatalf("create denied: %v", err)
		}
	}
	var cnt int64
	if err := db.Model(&dmValueLite{}).Where("name LIKE 'Device.FaultMgmt.%'").Count(&cnt).Error; err != nil {
		t.Fatalf("count denied: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expected 0 denied rows, got %d", cnt)
	}

	allowed := dmValueLite{Name: "Device.DeviceInfo.SerialNumber"}
	if !DenyByPrefixes(allowed.Name, rules[0].DenyPrefixes) {
		if err := db.Create(&allowed).Error; err != nil {
			t.Fatalf("create allowed: %v", err)
		}
	}
	if err := db.Model(&dmValueLite{}).Where("name = ?", "Device.DeviceInfo.SerialNumber").Count(&cnt).Error; err != nil {
		t.Fatalf("count allowed: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 allowed row, got %d", cnt)
	}
}

func TestMiddleware_GormCreateBatchFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&dmValueLite{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rules := RulesForPrefixDeny("tr069_datamodel_values", "Name", []string{"Device.FaultMgmt."})

	rows := []dmValueLite{
		{Name: "Device.FaultMgmt.A"},
		{Name: "Device.DeviceInfo.A"},
		{Name: "Device.FaultMgmt.B"},
	}
	if outAny, replaced := FilterForTable(rows, "tr069_datamodel_values", rules); replaced {
		out, ok := outAny.([]dmValueLite)
		if !ok {
			t.Fatalf("filter type: %T", outAny)
		}
		rows = out
	}
	if len(rows) > 0 {
		if err := db.Create(&rows).Error; err != nil {
			t.Fatalf("create batch: %v", err)
		}
	}
	var cnt int64
	if err := db.Model(&dmValueLite{}).Count(&cnt).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 total row, got %d", cnt)
	}
	if err := db.Model(&dmValueLite{}).Where("name LIKE 'Device.FaultMgmt.%'").Count(&cnt).Error; err != nil {
		t.Fatalf("count denied: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expected 0 denied rows, got %d", cnt)
	}
}
