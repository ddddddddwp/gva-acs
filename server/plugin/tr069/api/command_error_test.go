package api

import (
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
