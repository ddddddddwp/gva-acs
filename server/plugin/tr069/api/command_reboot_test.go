package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRebootIgnoresClientCommandKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DeviceRPCMethods),
		new(model.Command), new(model.CommandEvent),
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "API-REBOOT", LastInform: now}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := db.Create(&model.DeviceRPCMethods{
		DeviceID: device.ID, MethodsJSON: datatypes.JSON(`["Reboot"]`),
	}).Error; err != nil {
		t.Fatalf("create capabilities: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "deviceId", Value: strconv.Itoa(int(device.ID))}}
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/tr069/command/"+strconv.Itoa(int(device.ID))+"/reboot",
		strings.NewReader(`{"commandKey":"client-controlled"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	new(CommandApi).Reboot(context)

	var command model.Command
	if err := db.First(&command, "operation = ?", "Reboot").Error; err != nil {
		t.Fatalf("load Reboot command: %v", err)
	}
	if command.CommandKey == nil || *command.CommandKey == "client-controlled" ||
		!strings.HasPrefix(*command.CommandKey, "rpc-") {
		t.Fatalf("stored CommandKey = %v", command.CommandKey)
	}
	if string(command.ParamsJSON) != `{}` {
		t.Fatalf("stored params = %s, want {}", command.ParamsJSON)
	}
}
