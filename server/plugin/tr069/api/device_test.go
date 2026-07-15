package api

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	deviceResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDeviceResponseExposesRPCMethods(t *testing.T) {
	field, ok := reflect.TypeOf(deviceResponse.DeviceResponse{}).FieldByName("RPCMethods")
	if !ok {
		t.Fatal("DeviceResponse missing RPCMethods")
	}
	if got := field.Tag.Get("json"); got != "rpcMethods" {
		t.Fatalf("RPCMethods json tag = %q, want %q", got, "rpcMethods")
	}
}

func TestLoadDeviceRPCMethodsUsesOneBatchQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows := []model.DeviceRPCMethods{
		{DeviceID: 11, MethodsJSON: datatypes.JSON(`["GetRPCMethods","Reboot"]`)},
		{DeviceID: 12, MethodsJSON: datatypes.JSON(`["GetParameterValues"]`)},
		{DeviceID: 14, MethodsJSON: datatypes.JSON(`null`)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create method rows: %v", err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:count_rpc_method_queries", func(tx *gorm.DB) {
		if tx.Statement.Table == (model.DeviceRPCMethods{}).TableName() {
			queryCount++
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}

	got, err := loadDeviceRPCMethods(context.Background(), db, []uint{11, 12, 13, 14})
	if err != nil {
		t.Fatalf("loadDeviceRPCMethods() error = %v", err)
	}
	if queryCount != 1 {
		t.Fatalf("RPC method query count = %d, want 1", queryCount)
	}
	if want := []string{"GetRPCMethods", "Reboot"}; !reflect.DeepEqual(got[11], want) {
		t.Fatalf("methods[11] = %#v, want %#v", got[11], want)
	}
	if want := []string{"GetParameterValues"}; !reflect.DeepEqual(got[12], want) {
		t.Fatalf("methods[12] = %#v, want %#v", got[12], want)
	}
	if methods, ok := got[13]; !ok || methods == nil || len(methods) != 0 {
		t.Fatalf("methods[13] = %#v, present=%t; want a present empty slice", methods, ok)
	}
	if methods := got[14]; methods == nil || len(methods) != 0 {
		t.Fatalf("methods[14] = %#v, want non-nil empty slice", methods)
	}
	body, err := json.Marshal(deviceResponse.DeviceResponse{RPCMethods: got[14]})
	if err != nil {
		t.Fatalf("marshal DeviceResponse: %v", err)
	}
	if !strings.Contains(string(body), `"rpcMethods":[]`) {
		t.Fatalf("DeviceResponse JSON = %s, want rpcMethods:[]", body)
	}
}

func TestLoadDeviceRPCMethodsSkipsQueryForEmptyPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:count_empty_rpc_method_queries", func(*gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}

	got, err := loadDeviceRPCMethods(context.Background(), db, nil)
	if err != nil {
		t.Fatalf("loadDeviceRPCMethods() error = %v", err)
	}
	if len(got) != 0 || queryCount != 0 {
		t.Fatalf("loadDeviceRPCMethods(nil) = %#v, queries=%d; want empty map and zero queries", got, queryCount)
	}
}
