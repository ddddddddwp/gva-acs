package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCommandFailureMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "offline", err: service.ErrDeviceOffline, want: "设备离线，无法下发任务"},
		{name: "queue unavailable", err: service.ErrCommandQueueUnavailable, want: "命令队列不可用"},
		{name: "other", err: errors.New("boom"), want: "任务下发失败: boom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandFailureMessage(tt.err); got != tt.want {
				t.Fatalf("commandFailureMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommandAPIWakeupFailureReturnsStableFailedSubmitResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate command API models: %v", err)
	}
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "API-WAKE-FAIL", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	previousService := commandService
	commandService = service.NewCommandService(service.NewCommandManager(db, func(context.Context, string) error {
		return errors.New("redis unavailable")
	}, service.WithCommandManagerNow(func() time.Time { return now })))
	t.Cleanup(func() { commandService = previousService })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tr069/command/:deviceId/getRPCMethods", new(CommandApi).SyncRPCMethods)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tr069/command/%d/getRPCMethods", device.ID), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var response struct {
		Code int                  `json:"code"`
		Data service.SubmitResult `json:"data"`
		Msg  string               `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode API response %q: %v", recorder.Body.String(), err)
	}
	if response.Code != 0 || response.Data.CommandID == "" || response.Data.Status != model.CommandStatusFailed {
		t.Fatalf("API response = code:%d data:%#v, want success envelope with FAILED result", response.Code, response.Data)
	}
	if response.Msg != "任务已持久化，但设备唤醒调度失败" {
		t.Fatalf("API message = %q, want durable wakeup-failure semantics", response.Msg)
	}
}

func TestCommandAPISubmitsEveryTypedRPCOperation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate command API models: %v", err)
	}
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "API-ALL-RPC", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	methods, err := json.Marshal([]string{
		"GetParameterValues", "GetParameterNames", "GetParameterAttributes",
		"SetParameterValues", "SetParameterAttributes", "AddObject", "DeleteObject",
		"Download", "Upload", "Reboot", "FactoryReset",
	})
	if err != nil {
		t.Fatalf("encode capabilities: %v", err)
	}
	if err := db.Create(&model.DeviceRPCMethods{DeviceID: device.ID, MethodsJSON: datatypes.JSON(methods)}).Error; err != nil {
		t.Fatalf("create device capabilities: %v", err)
	}

	previousService := commandService
	commandService = service.NewCommandService(service.NewCommandManager(db, func(context.Context, string) error {
		return nil
	}, service.WithCommandManagerNow(func() time.Time { return now })))
	t.Cleanup(func() { commandService = previousService })

	api := new(CommandApi)
	tests := []struct {
		operation string
		path      string
		handler   gin.HandlerFunc
		body      string
	}{
		{operation: "GetRPCMethods", path: "getRPCMethods", handler: api.SyncRPCMethods},
		{operation: "GetParameterValues", path: "getParameterValues", handler: api.GetParameterValues, body: `{"paths":["Device."]}`},
		{operation: "GetParameterNames", path: "getParameterNames", handler: api.GetParameterNames, body: `{"parameterPath":"Device.","nextLevel":true}`},
		{operation: "GetParameterAttributes", path: "getParameterAttributes", handler: api.GetParameterAttributes, body: `{"parameterNames":["Device.DeviceInfo.Manufacturer"]}`},
		{operation: "SetParameterValues", path: "setParameterValues", handler: api.SetParameterValues, body: `{"parameterKey":"set-1","parameters":[{"name":"Device.Test.Value","type":"xsd:string","value":"new"}]}`},
		{operation: "SetParameterAttributes", path: "setParameterAttributes", handler: api.SetParameterAttributes, body: `{"parameterAttributes":[{"name":"Device.Test.Value","notificationChange":true,"notification":2,"accessListChange":false,"accessList":[]}]}`},
		{operation: "AddObject", path: "addObject", handler: api.AddObject, body: `{"objectName":"Device.WiFi.SSID.","parameterKey":"add-1"}`},
		{operation: "DeleteObject", path: "deleteObject", handler: api.DeleteObject, body: `{"objectName":"Device.WiFi.SSID.7.","parameterKey":"delete-1"}`},
		{operation: "Download", path: "download", handler: api.Download, body: `{"fileType":"1 Firmware Upgrade Image","url":"https://example.test/firmware.bin","fileSize":1024,"targetFileName":"firmware.bin","delaySeconds":0}`},
		{operation: "Upload", path: "upload", handler: api.Upload, body: `{"fileType":"1 Vendor Configuration File","url":"https://example.test/upload","delaySeconds":0}`},
		{operation: "Reboot", path: "reboot", handler: api.Reboot, body: `{"commandKey":"reboot-1"}`},
		{operation: "FactoryReset", path: "factoryReset", handler: api.FactoryReset},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			router := gin.New()
			router.POST("/tr069/command/:deviceId/"+tt.path, tt.handler)
			request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tr069/command/%d/%s", device.ID, tt.path), bytes.NewBufferString(tt.body))
			if tt.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			var response struct {
				Code int                  `json:"code"`
				Data service.SubmitResult `json:"data"`
				Msg  string               `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode API response %q: %v", recorder.Body.String(), err)
			}
			if response.Code != 0 || response.Data.CommandID == "" {
				t.Fatalf("API response = code:%d data:%#v msg:%q, want submitted command", response.Code, response.Data, response.Msg)
			}

			var command model.Command
			if err := db.First(&command, "command_id = ?", response.Data.CommandID).Error; err != nil {
				t.Fatalf("load submitted command: %v", err)
			}
			if command.Operation != tt.operation {
				t.Fatalf("persisted operation = %q, want %q", command.Operation, tt.operation)
			}
		})
	}
}
