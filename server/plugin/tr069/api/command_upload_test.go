package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCommandAPIUploadUsesConfiguredIngressAndPersistsNoCredentials(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent), new(model.TransferTask), new(model.TransferEvent)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	device := model.Device{OUI: "8CE468", SerialNumber: "BS-UPLOAD-API", LastInform: time.Now().UTC()}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := db.Create(&model.DeviceRPCMethods{DeviceID: device.ID, MethodsJSON: datatypes.JSON(`["Upload"]`)}).Error; err != nil {
		t.Fatalf("create capabilities: %v", err)
	}
	previousRuntime := config.CurrentRuntime()
	previousService := commandService
	t.Cleanup(func() {
		config.StoreRuntime(previousRuntime.Settings)
		commandService = previousService
	})
	config.StoreRuntime(config.TR069Config{FileIngress: config.FileIngressConfig{
		Enabled: true, PublicBaseURL: "http://gva:7458",
		Authentication: config.FileIngressAuthConfig{Username: "configured-log-user", Password: "configured-log-password", Realm: "GVA", Schemes: []string{"basic"}},
		Channels:       map[string]config.TransferChannelConfig{"log": {Enabled: true, Path: "/acs/log"}},
	}})
	codec := adapter.NewCompositeCommandPayloadCodec(adapter.LogUploadPayloadCodec{})
	commandService = service.NewCommandService(service.NewCommandManager(db, func(context.Context, string) error { return nil },
		service.WithCommandPayloadProtector(codec), service.WithCommandCreatedHook(service.NewActiveUploadTaskHook(nil)),
	))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tr069/command/:deviceId/upload", new(CommandApi).Upload)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tr069/command/%d/upload", device.ID), strings.NewReader(`{"fileType":"Vendor Log File","delaySeconds":3}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || result.Code != 0 {
		t.Fatalf("response=%s decodeErr=%v", recorder.Body.String(), err)
	}
	var command model.Command
	if err := db.First(&command, "operation = ?", "Upload").Error; err != nil {
		t.Fatalf("load Upload command: %v", err)
	}
	persisted := string(command.ParamsJSON)
	for _, secret := range []string{"configured-log-user", "configured-log-password"} {
		if strings.Contains(persisted, secret) {
			t.Fatalf("persisted Upload params leaked %q: %s", secret, persisted)
		}
	}
	if !strings.Contains(persisted, "__GVA_TR069_LOG_UPLOAD_USERNAME__") || !strings.Contains(persisted, "__GVA_TR069_LOG_UPLOAD_PASSWORD__") {
		t.Fatalf("persisted Upload params missing placeholders: %s", persisted)
	}
	if !strings.Contains(persisted, "http://gva:7458/acs/log") {
		t.Fatalf("persisted Upload params missing configured URL: %s", persisted)
	}
	var tasks int64
	db.Model(new(model.TransferTask)).Where("command_id = ?", command.CommandID).Count(&tasks)
	if tasks != 1 {
		t.Fatalf("active tasks=%d", tasks)
	}
}
