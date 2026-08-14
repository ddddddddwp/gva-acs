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

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	commandResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCommandRecordAPIListHidesDetailPayloadAndDetailReturnsExactXML(t *testing.T) {
	db, device, command := seedCommandRecordAPI(t)
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	recordAPI := new(CommandRecordApi)
	router.GET("/tr069/command-record/list", recordAPI.List)
	router.GET("/tr069/command-record/:commandId", recordAPI.Detail)

	listRequest := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tr069/command-record/list?page=1&pageSize=10&deviceId=%d&status=FAILED", device.ID), nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if strings.Contains(listRecorder.Body.String(), "top-secret") || strings.Contains(listRecorder.Body.String(), "cwmp:GetParameterValues") {
		t.Fatalf("list response leaked detail-only payload: %s", listRecorder.Body.String())
	}
	var listResponse struct {
		Code int `json:"code"`
		Data struct {
			List  []commandResponse.CommandRecordSummary `json:"list"`
			Total int64                                  `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResponse.Code != 0 || listResponse.Data.Total != 1 || len(listResponse.Data.List) != 1 {
		t.Fatalf("list response = %#v, want one record", listResponse)
	}
	if listResponse.Data.List[0].CommandID != command.CommandID {
		t.Fatalf("list command ID = %q, want %q", listResponse.Data.List[0].CommandID, command.CommandID)
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/tr069/command-record/"+command.CommandID, nil)
	detailRecorder := httptest.NewRecorder()
	router.ServeHTTP(detailRecorder, detailRequest)
	var detailResponse struct {
		Code int                                 `json:"code"`
		Data commandResponse.CommandRecordDetail `json:"data"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailResponse); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailResponse.Code != 0 || len(detailResponse.Data.Events) < 3 || len(detailResponse.Data.XML) != 3 {
		t.Fatalf("detail response = %#v, want command events and XML", detailResponse)
	}
	if got := detailResponse.Data.XML[0].XML; got != `<cwmp:GetParameterValues id="42"/>` {
		t.Fatalf("outbound XML = %q, want exact UTF-8 payload", got)
	}
	if got := detailResponse.Data.XML[1].XML; got != `<cwmp:GetParameterValuesResponse id="42"/>` {
		t.Fatalf("inbound XML = %q, want exact UTF-8 payload", got)
	}
	detailBody := detailRecorder.Body.String()
	for _, secret := range []string{
		"__GVA_TR069_CONNECTION_REQUEST_PASSWORD__",
		"ciphertext-v1:legacy-result",
		"ciphertext-v1:legacy-event",
		"base64-legacy-ciphertext",
		"legacy-xml-secret",
	} {
		if strings.Contains(detailBody, secret) {
			t.Fatalf("detail response leaked protected value %q: %s", secret, detailBody)
		}
	}
	if !strings.Contains(detailBody, "******") {
		t.Fatalf("detail response did not contain redaction marker: %s", detailBody)
	}
	if !strings.Contains(detailBody, "top-secret") {
		t.Fatalf("unrelated password was changed: %s", detailBody)
	}
	if strings.Contains(listRecorder.Body.String(), "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__") ||
		strings.Contains(listRecorder.Body.String(), "ciphertext-v1:") {
		t.Fatalf("list response leaked protected command text: %s", listRecorder.Body.String())
	}
}

func TestCommandRecordAPIRetryCreatesLinkedCommand(t *testing.T) {
	db, _, original := seedCommandRecordAPI(t)
	previousDB := global.GVA_DB
	previousService := commandService
	previousAuthorizer := authorizeCommandRetry
	global.GVA_DB = db
	commandService = service.NewCommandService(service.NewCommandManager(db, func(context.Context, string) error { return nil }))
	authorizeCommandRetry = func(*gin.Context, model.Command) bool { return true }
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		commandService = previousService
		authorizeCommandRetry = previousAuthorizer
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tr069/command-record/:commandId/retry", new(CommandRecordApi).Retry)
	request := httptest.NewRequest(http.MethodPost, "/tr069/command-record/"+original.CommandID+"/retry", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var response struct {
		Code int                  `json:"code"`
		Data service.SubmitResult `json:"data"`
		Msg  string               `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode retry response: %v", err)
	}
	if response.Code != 0 || response.Data.CommandID == "" || response.Data.CommandID == original.CommandID {
		t.Fatalf("retry response = %#v, want a new linked command", response)
	}
	var retry model.Command
	if err := db.First(&retry, "command_id = ?", response.Data.CommandID).Error; err != nil {
		t.Fatalf("load retry command: %v", err)
	}
	if retry.RetryOf != original.CommandID || retry.Operation != original.Operation {
		t.Fatalf("retry lineage/operation = %q/%q, want %q/%q", retry.RetryOf, retry.Operation, original.CommandID, original.Operation)
	}
}

func TestCommandRecordAPIRetryRequiresOriginalOperationPermission(t *testing.T) {
	db, _, original := seedCommandRecordAPI(t)
	previousDB := global.GVA_DB
	previousAuthorizer := authorizeCommandRetry
	global.GVA_DB = db
	authorizeCommandRetry = func(*gin.Context, model.Command) bool { return false }
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		authorizeCommandRetry = previousAuthorizer
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tr069/command-record/:commandId/retry", new(CommandRecordApi).Retry)
	request := httptest.NewRequest(http.MethodPost, "/tr069/command-record/"+original.CommandID+"/retry", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode denied retry response: %v", err)
	}
	if response.Code == 0 || response.Msg != "缺少原操作的下发权限" {
		t.Fatalf("denied retry response = %#v", response)
	}
	var count int64
	if err := db.Model(new(model.Command)).Count(&count).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	if count != 1 {
		t.Fatalf("command count after denied retry = %d, want original only", count)
	}
}

func seedCommandRecordAPI(t *testing.T) (*gorm.DB, model.Device, model.Command) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods), new(model.Command), new(model.CommandEvent), new(model.CommandXML)); err != nil {
		t.Fatalf("migrate command record models: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	device := model.Device{OUI: "001122", SerialNumber: "RECORD-API", LastInform: now}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	methods, _ := json.Marshal([]string{"GetParameterValues"})
	if err := db.Create(&model.DeviceRPCMethods{DeviceID: device.ID, MethodsJSON: datatypes.JSON(methods)}).Error; err != nil {
		t.Fatalf("create capabilities: %v", err)
	}
	finishedAt := now
	command := model.Command{
		CommandID:    "record-command-1",
		DeviceID:     device.ID,
		DeviceKey:    "001122-RECORD-API",
		Operation:    "GetParameterValues",
		ParamsJSON:   model.LongTextJSON(`{"paths":["Device."],"parameters":[{"name":"Device.ManagementServer.ConnectionRequestPassword","value":"__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"},{"name":"Download.Password","value":"top-secret"}]}`),
		ResultJSON:   model.LongTextJSON(`{"parameters":[{"name":"Device.ManagementServer.ConnectionRequestPassword","value":"ciphertext-v1:legacy-result"}],"fault":"sample"}`),
		Status:       model.CommandStatusFailed,
		QueuedAt:     now.Add(-time.Minute),
		FinishedAt:   &finishedAt,
		FailureStage: "cwmp.fault",
		FaultCode:    9002,
		FaultString:  "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__",
		CreatedAt:    now.Add(-time.Minute),
		UpdatedAt:    now,
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatalf("create command: %v", err)
	}
	events := []model.CommandEvent{
		{CommandID: command.CommandID, EventType: model.CommandEventCreated, ToStatus: model.CommandStatusWaitingDevice, CreatedAt: now.Add(-time.Minute)},
		{CommandID: command.CommandID, EventType: "COMMAND_FAILED", FromStatus: model.CommandStatusSent, ToStatus: model.CommandStatusFailed, Stage: "cwmp.fault", Message: "ciphertext-v1:legacy-event", PayloadJSON: model.LongTextJSON(`"__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"`), CreatedAt: now},
		{CommandID: command.CommandID, EventType: "LEGACY_PAYLOAD", Stage: "legacy", Message: "unrelated event", PayloadJSON: model.LongTextJSON(`{"password_ciphertext":"base64-legacy-ciphertext","password":"download-secret"}`), CreatedAt: now},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatalf("create events: %v", err)
	}
	xml := []model.CommandXML{
		{CommandID: command.CommandID, Direction: "outbound", Method: command.Operation, CWMPID: "42", Payload: []byte(`<cwmp:GetParameterValues id="42"/>`), CreatedAt: now.Add(-time.Second)},
		{CommandID: command.CommandID, Direction: "inbound", Method: command.Operation + "Response", CWMPID: "42", Payload: []byte(`<cwmp:GetParameterValuesResponse id="42"/>`), CreatedAt: now},
		{CommandID: command.CommandID, Direction: "outbound", Method: "SetParameterValues", CWMPID: "43", Payload: []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>legacy-xml-secret</Value></ParameterValueStruct></Envelope>`), CreatedAt: now},
	}
	if err := db.Create(&xml).Error; err != nil {
		t.Fatalf("create XML records: %v", err)
	}
	return db, device, command
}
