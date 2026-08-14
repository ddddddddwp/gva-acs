package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	deviceResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeDeviceDeletion struct {
	result   service.DeviceDeletionResult
	err      error
	deviceID uint
	calls    int
}

func (f *fakeDeviceDeletion) Delete(_ context.Context, deviceID uint) (service.DeviceDeletionResult, error) {
	f.calls++
	f.deviceID = deviceID
	return f.result, f.err
}

func TestConnectionProfileAPIStoresManualOverrideWithoutExposingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	device := model.Device{OUI: "001122", SerialNumber: "PROFILE-API"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	previousDB := global.GVA_DB
	previousRuntime := tr069Config.CurrentRuntime()
	global.GVA_DB = db
	tr069Config.StoreRuntime(tr069Config.TR069Config{ConnectionRequest: tr069Config.ConnectionRequestConfig{
		CredentialKeyVersion:    "v1",
		CredentialEncryptionKey: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x71}, 32)),
	}})
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		tr069Config.StoreRuntime(previousRuntime.Settings)
	})

	engine := gin.New()
	api := new(DeviceApi)
	engine.PUT("/tr069/device/:deviceId/connection-profile", api.UpdateConnectionProfile)
	engine.GET("/tr069/device/:deviceId/connection-profile", api.GetConnectionProfile)
	payload := []byte(`{"overrideUrl":"http://127.0.0.1:8400","username":"manual-user","password":"manual-secret"}`)
	profilePath := "/tr069/device/" + strconv.FormatUint(uint64(device.ID), 10) + "/connection-profile"
	update := httptest.NewRequest(http.MethodPut, profilePath, bytes.NewReader(payload))
	update.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	engine.ServeHTTP(updateRecorder, update)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	for _, forbidden := range []string{"manual-secret", "passwordCiphertext", "credentialKeyVersion"} {
		if strings.Contains(updateRecorder.Body.String(), forbidden) {
			t.Fatalf("update response leaked %q: %s", forbidden, updateRecorder.Body.String())
		}
	}

	var profile model.ConnectionProfile
	if err := db.First(&profile, "device_id = ?", device.ID).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.OverrideURL != "http://127.0.0.1:8400" || profile.Username != "manual-user" || profile.CredentialSource != model.ConnectionCredentialSourceManual {
		t.Fatalf("profile=%#v", profile)
	}
	if len(profile.PasswordCiphertext) == 0 || bytes.Contains(profile.PasswordCiphertext, []byte("manual-secret")) {
		t.Fatal("manual password was not encrypted")
	}

	getRecorder := httptest.NewRecorder()
	engine.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, profilePath, nil))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	for _, forbidden := range []string{"manual-secret", "passwordCiphertext", "credentialKeyVersion"} {
		if strings.Contains(getRecorder.Body.String(), forbidden) {
			t.Fatalf("get response leaked %q: %s", forbidden, getRecorder.Body.String())
		}
	}
}

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

func TestDeleteDeviceDelegatesToCascadeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deletion := &fakeDeviceDeletion{result: service.DeviceDeletionResult{DeviceID: 42, DeletedObjects: 2}}
	engine := gin.New()
	engine.DELETE("/tr069/device/:deviceId", NewDeviceApi(deletion).DeleteDevice)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/tr069/device/42", nil))
	if deletion.calls != 1 || deletion.deviceID != 42 {
		t.Fatalf("deletion calls/id = %d/%d", deletion.calls, deletion.deviceID)
	}
	if !strings.Contains(recorder.Body.String(), `"code":0`) || !strings.Contains(recorder.Body.String(), `"删除成功"`) {
		t.Fatalf("delete response = %s", recorder.Body.String())
	}
}

func TestDeleteDeviceReturnsSafeStageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deletion := &fakeDeviceDeletion{err: &service.DeviceDeletionError{Stage: service.DeletionStageArtifacts, Cause: errors.New("secret MinIO endpoint")}}
	engine := gin.New()
	engine.DELETE("/tr069/device/:deviceId", NewDeviceApi(deletion).DeleteDevice)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/tr069/device/7", nil))
	if !strings.Contains(recorder.Body.String(), service.DeletionStageArtifacts) || strings.Contains(recorder.Body.String(), "secret MinIO endpoint") {
		t.Fatalf("unsafe deletion response = %s", recorder.Body.String())
	}
}

func TestDeleteDeviceMapsMissingAndInvalidIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deletion := &fakeDeviceDeletion{err: service.ErrDeviceNotFound}
	engine := gin.New()
	engine.DELETE("/tr069/device/:deviceId", NewDeviceApi(deletion).DeleteDevice)
	for _, path := range []string{"/tr069/device/999", "/tr069/device/not-a-number"} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, path, nil))
		if !strings.Contains(recorder.Body.String(), `"code":7`) {
			t.Fatalf("%s response = %s", path, recorder.Body.String())
		}
	}
	if deletion.calls != 1 {
		t.Fatalf("deletion calls = %d, want only valid ID call", deletion.calls)
	}
}
