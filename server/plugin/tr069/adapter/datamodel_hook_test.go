package adapter

import (
	"context"
	"strings"
	"testing"
	"time"

	appGlobal "github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type captureIngest struct {
	cmds []*core.Command
}

type transferLifecycleSpy struct {
	uploadCommandID string
	uploadStatus    int
	commandKey      string
	faultCode       int
	faultString     string
}

func (s *transferLifecycleSpy) OnUploadResponse(_ context.Context, commandID string, status int, _ time.Time) error {
	s.uploadCommandID = commandID
	s.uploadStatus = status
	return nil
}

func (s *transferLifecycleSpy) OnTransferComplete(_ context.Context, commandKey string, faultCode int, faultString string, _ time.Time) error {
	s.commandKey = commandKey
	s.faultCode = faultCode
	s.faultString = faultString
	return nil
}

func TestDataModelHookForwardsUploadResponseAndTransferComplete(t *testing.T) {
	inflight := NewMemoryInflightRepo(10 * time.Minute)
	ctx := context.Background()
	if err := inflight.Save(ctx, core.InflightRequest{DeviceKey: "8CE468-BS-HOOK", CommandID: "upload-command", CwmpID: "cwmp-upload", RequestName: tr069.MethodUpload}); err != nil {
		t.Fatalf("save inflight: %v", err)
	}
	spy := new(transferLifecycleSpy)
	hook := NewDataModelHook(nil, inflight, nil, WithDataModelHookTransferLifecycle(spy), WithDataModelHookNow(func() time.Time {
		return time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)
	}))
	session := &core.Session{DeviceKey: "8CE468-BS-HOOK"}
	handled, err := hook.OnResponse(ctx, session, &tr069.Message{Method: tr069.MethodUploadResponse, ID: "cwmp-upload", Status: 1})
	if err != nil || !handled || spy.uploadCommandID != "upload-command" || spy.uploadStatus != 1 {
		t.Fatalf("handled=%v spy=%#v err=%v", handled, spy, err)
	}
	handled, err = hook.OnTransferComplete(ctx, session, &tr069.Message{Method: tr069.MethodTransferComplete, CommandKey: "command-key", TransferFaultCode: 9010, TransferFaultString: "transfer failed"})
	if err != nil || !handled || spy.commandKey != "command-key" || spy.faultCode != 9010 || spy.faultString != "transfer failed" {
		t.Fatalf("transfer handled=%v spy=%#v err=%v", handled, spy, err)
	}
}

func (i *captureIngest) Enqueue(ctx context.Context, cmd *core.Command) error {
	_ = ctx
	if cmd != nil {
		i.cmds = append(i.cmds, cmd)
	}
	return nil
}

func TestDataModelHookGPVCollectsConnectionProfile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	previousDB := appGlobal.GVA_DB
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = previousDB })
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	device := model.Device{OUI: "001122", SerialNumber: "GPV-PROFILE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	cipher, err := NewCredentialCipher(testCredentialConfig(0x62))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	profiles := NewConnectionProfileRepository(db, cipher)
	hook := NewDataModelHook(nil, nil, nil, WithDataModelHookProfiles(profiles))

	if err := hook.persistGPV(context.Background(), device.ID, []tr069.Parameter{
		{Name: connectionRequestURLName, Value: "http://127.0.0.1:8400", Type: "xsd:string"},
		{Name: connectionRequestUsernameName, Value: "gpv-user", Type: "xsd:string"},
	}); err != nil {
		t.Fatalf("persist GPV: %v", err)
	}

	profile := loadConnectionProfile(t, db, device.ID)
	if profile.DiscoveredURL != "http://127.0.0.1:8400" || profile.Username != "gpv-user" {
		t.Fatalf("profile=%#v", profile)
	}
}

func TestDataModelHookProvisionerSchedulesNeededGPVProfile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	previousDB := appGlobal.GVA_DB
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = previousDB })
	if err := db.AutoMigrate(new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	device := model.Device{OUI: "001122", SerialNumber: "GPV-SCHEDULE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	cipher, err := NewCredentialCipher(testCredentialConfig(0x64))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	scheduler := &recordingCredentialScheduler{accept: true}
	hook := NewDataModelHook(nil, nil, nil,
		WithDataModelHookProfiles(NewConnectionProfileRepository(db, cipher)),
		WithDataModelHookProvisioner(scheduler),
	)
	if err := hook.persistGPV(context.Background(), device.ID, []tr069.Parameter{
		{Name: connectionRequestURLName, Value: "http://127.0.0.1:8400", Type: "xsd:string"},
	}); err != nil {
		t.Fatalf("persist GPV: %v", err)
	}
	if len(scheduler.deviceIDs) != 1 || scheduler.deviceIDs[0] != device.ID {
		t.Fatalf("scheduled device IDs = %#v, want [%d]", scheduler.deviceIDs, device.ID)
	}
}

func TestDataModelHook_PersistsGPVAndExpandsGPN(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = nil })

	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.DataModelValue), new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dev := model.Device{OUI: "001122", SerialNumber: "1234567890"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	inflight := NewMemoryInflightRepo(10 * time.Minute)
	ingest := &captureIngest{}
	hook := NewDataModelHook(nil, inflight, ingest, WithDataModelHookNow(func() time.Time { return time.Unix(10, 0) }))

	ctx := context.Background()
	session := &core.Session{
		DeviceKey: "001122-1234567890",
		DeviceID:  "1234567890",
	}

	if err := inflight.Save(ctx, core.InflightRequest{
		DeviceKey:   session.DeviceKey,
		CommandID:   "cmd-gpv",
		CwmpID:      "cwmp-gpv",
		RequestName: tr069.MethodGetParameterValues,
		SentAt:      time.Now(),
	}); err != nil {
		t.Fatalf("save inflight: %v", err)
	}

	handled, err := hook.OnResponse(ctx, session, &tr069.Message{
		Method: tr069.MethodGetParameterValuesResponse,
		ID:     "cwmp-gpv",
		Parameters: []tr069.Parameter{
			{Name: "Device.DeviceInfo.SerialNumber", Value: "1234567890", Type: "xsd:string"},
		},
	})
	if err != nil {
		t.Fatalf("OnResponse error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true for correlated response")
	}

	var got model.DataModelValue
	if err := db.Where("device_id = ? AND name = ?", dev.ID, "Device.DeviceInfo.SerialNumber").First(&got).Error; err != nil {
		t.Fatalf("expected datamodel value persisted: %v", err)
	}

	if err := db.Create(&model.Command{
		CommandID:  "cmd-gpn",
		DeviceKey:  session.DeviceKey,
		Operation:  "GetParameterNames",
		ParamsJSON: model.LongTextJSON(`{"parameterPath":"Device.","depth":0,"maxDepth":2}`),
		Status:     "SENDING",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error; err != nil {
		t.Fatalf("create cmd: %v", err)
	}

	if err := inflight.Save(ctx, core.InflightRequest{
		DeviceKey:   session.DeviceKey,
		CommandID:   "cmd-gpn",
		CwmpID:      "cwmp-gpn",
		RequestName: tr069.MethodGetParameterNames,
		SentAt:      time.Now(),
	}); err != nil {
		t.Fatalf("save inflight gpn: %v", err)
	}

	handled, err = hook.OnResponse(ctx, session, &tr069.Message{
		Method: tr069.MethodGetParameterNamesResponse,
		ID:     "cwmp-gpn",
		ParameterInfos: []tr069.ParameterInfo{
			{Name: "Device.Services.", Writable: false},
			{Name: "Device.DeviceInfo.SerialNumber", Writable: false},
		},
	})
	if err != nil {
		t.Fatalf("OnResponse gpn error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true for correlated gpn response")
	}
	if len(ingest.cmds) == 0 {
		t.Fatalf("expected expansion enqueued commands")
	}
	foundGPN := false
	foundGPV := false
	for _, c := range ingest.cmds {
		if c.Operation == "GetParameterNames" {
			foundGPN = true
		}
		if c.Operation == "GetParameterValues" {
			foundGPV = true
		}
	}
	if !foundGPN || !foundGPV {
		t.Fatalf("expected both GetParameterNames and GetParameterValues enqueued, got=%v", ingest.cmds)
	}
}

// TestDataModelHook_PersistsGPVWithXsiType 测试解析带有 xsi:type 属性的参数并存储到数据库
func TestDataModelHook_PersistsGPVWithXsiType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = nil })

	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.DataModelValue), new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dev := model.Device{OUI: "001122", SerialNumber: "1234567890"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	inflight := NewMemoryInflightRepo(10 * time.Minute)
	ingest := &captureIngest{}
	hook := NewDataModelHook(nil, inflight, ingest, WithDataModelHookNow(func() time.Time { return time.Unix(10, 0) }))

	ctx := context.Background()
	session := &core.Session{
		DeviceKey: "001122-1234567890",
		DeviceID:  "1234567890",
	}

	// 保存 inflight 请求
	if err := inflight.Save(ctx, core.InflightRequest{
		DeviceKey:   session.DeviceKey,
		CommandID:   "cmd-gpv",
		CwmpID:      "cwmp-gpv",
		RequestName: tr069.MethodGetParameterValues,
		SentAt:      time.Now(),
	}); err != nil {
		t.Fatalf("save inflight: %v", err)
	}

	// 模拟解析后的参数列表（这些参数类型应该与解析 xsi:type 后的一致）
	params := []tr069.Parameter{
		{Name: "Device.DeviceInfo.SerialNumber", Value: "SN123456789", Type: "xsd:string"},
		{Name: "Device.DeviceInfo.MU.1.Slot.1.SoftwareVersion", Value: "5.1.0.r62694M", Type: "xsd:string"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.18.0.20:7547/", Type: "xsd:string"},
		{Name: "Device.DeviceInfo.SomeBoolean", Value: "true", Type: "xsd:boolean"},
		{Name: "Device.DeviceInfo.SomeInt", Value: "-12345", Type: "xsd:int"},
		{Name: "Device.DeviceInfo.SomeLong", Value: "9876543210", Type: "xsd:long"},
		{Name: "Device.DeviceInfo.SomeUnsignedInt", Value: "65535", Type: "xsd:unsignedInt"},
		{Name: "Device.DeviceInfo.SomeUnsignedLong", Value: "18446744073709551615", Type: "xsd:unsignedLong"},
		{Name: "Device.DeviceInfo.LastUpdated", Value: "2026-07-16T02:30:00Z", Type: "xsd:dateTime"},
	}
	expected := map[string]struct {
		valueType string
		valueJSON string
	}{
		"Device.DeviceInfo.SerialNumber":                {valueType: "string", valueJSON: `"SN123456789"`},
		"Device.DeviceInfo.MU.1.Slot.1.SoftwareVersion": {valueType: "string", valueJSON: `"5.1.0.r62694M"`},
		"Device.ManagementServer.ConnectionRequestURL":  {valueType: "string", valueJSON: `"http://172.18.0.20:7547/"`},
		"Device.DeviceInfo.SomeBoolean":                 {valueType: "boolean", valueJSON: `true`},
		"Device.DeviceInfo.SomeInt":                     {valueType: "int", valueJSON: `-12345`},
		"Device.DeviceInfo.SomeLong":                    {valueType: "long", valueJSON: `9876543210`},
		"Device.DeviceInfo.SomeUnsignedInt":             {valueType: "unsignedInt", valueJSON: `65535`},
		"Device.DeviceInfo.SomeUnsignedLong":            {valueType: "unsignedLong", valueJSON: `18446744073709551615`},
		"Device.DeviceInfo.LastUpdated":                 {valueType: "dateTime", valueJSON: `"2026-07-16T02:30:00Z"`},
	}

	handled, err := hook.OnResponse(ctx, session, &tr069.Message{
		Method:     tr069.MethodGetParameterValuesResponse,
		ID:         "cwmp-gpv",
		Parameters: params,
	})
	if err != nil {
		t.Fatalf("OnResponse error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true for correlated response")
	}

	var storageRows []struct {
		Name         string
		StorageClass string
	}
	if err := db.Raw("SELECT name, typeof(value_json) AS storage_class FROM tr069_datamodel_values WHERE device_id = ?", dev.ID).Scan(&storageRows).Error; err != nil {
		t.Fatalf("inspect value_json storage: %v", err)
	}
	for _, row := range storageRows {
		if row.StorageClass != "text" && row.StorageClass != "blob" {
			t.Errorf("value_json storage class for %s = %s, want text or blob", row.Name, row.StorageClass)
		}
	}

	// 验证数据是否正确存储到数据库
	var values []model.DataModelValue
	if err := db.Where("device_id = ?", dev.ID).Find(&values).Error; err != nil {
		t.Fatalf("expected datamodel values persisted: %v", err)
	}

	if len(values) != len(params) {
		t.Fatalf("expected %d values, got %d", len(params), len(values))
	}

	// 验证每个参数的类型和值
	for _, value := range values {
		want, ok := expected[value.Name]
		if !ok {
			t.Errorf("unexpected parameter persisted: %s", value.Name)
			continue
		}
		if value.ValueType != want.valueType {
			t.Errorf("value_type for %s = %q, want %q", value.Name, value.ValueType, want.valueType)
		}
		if got := string(value.ValueJSON); got != want.valueJSON {
			t.Errorf("value_json for %s = %q, want %q", value.Name, got, want.valueJSON)
		}
		delete(expected, value.Name)
	}
	for name := range expected {
		t.Errorf("expected parameter %s to be persisted", name)
	}
}

// TestDataModelHook_PersistGPV_DebugValueType 专门测试 value_type 字段存储
func TestDataModelHook_PersistGPV_DebugValueType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = nil })

	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.DataModelValue), new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dev := model.Device{OUI: "001122", SerialNumber: "1234567890"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	// 直接调用 persistGPV 进行测试
	inflight := NewMemoryInflightRepo(10 * time.Minute)
	ingest := &captureIngest{}
	hook := NewDataModelHook(nil, inflight, ingest, WithDataModelHookNow(func() time.Time { return time.Unix(10, 0) }))

	// 测试不同类型的参数
	params := []tr069.Parameter{
		{Name: "Device.Test.String", Value: "hello", Type: "xsd:string"},
		{Name: "Device.Test.Boolean", Value: "true", Type: "xsd:boolean"},
		{Name: "Device.Test.Int", Value: "123", Type: "xsd:int"},
		{Name: "Device.Test.Long", Value: "456789", Type: "xsd:long"},
		{Name: "Device.Test.UnsignedInt", Value: "999", Type: "xsd:unsignedInt"},
		{Name: "Device.Test.EmptyType", Value: "test", Type: ""}, // 空类型
	}

	// 直接调用 persistGPV
	err = hook.persistGPV(context.Background(), dev.ID, params)
	if err != nil {
		t.Fatalf("persistGPV error: %v", err)
	}

	// 查询数据库，验证 value_type 是否正确存储（使用 raw query 避免 JSON 解析问题）
	type result struct {
		ID        uint
		Name      string
		ValueType string
		ValueJSON string
	}

	var results []result
	if err := db.Raw("SELECT id, name, value_type, CAST(value_json AS TEXT) as value_json FROM tr069_datamodel_values WHERE device_id = ?", dev.ID).Scan(&results).Error; err != nil {
		t.Fatalf("query error: %v", err)
	}

	t.Logf("Total records in database: %d", len(results))

	// 验证每个记录
	for _, r := range results {
		t.Logf("Name: %s, ValueType: '%s', ValueJSON: %s", r.Name, r.ValueType, r.ValueJSON)
	}

	// 详细断言
	for _, param := range params {
		var found result
		err := db.Raw("SELECT id, name, value_type, CAST(value_json AS TEXT) as value_json FROM tr069_datamodel_values WHERE device_id = ? AND name = ?", dev.ID, param.Name).Scan(&found).Error
		if err != nil {
			t.Errorf("not found: %s, error: %v", param.Name, err)
			continue
		}

		// 验证 value_type 字段
		expectedType := param.Type
		if param.Type != "" && strings.Contains(param.Type, ":") {
			// xsd:string -> string
			expectedType = param.Type[strings.Index(param.Type, ":")+1:]
		}

		if found.ValueType != expectedType {
			t.Errorf("value_type mismatch for %s: expected '%s', got '%s'", param.Name, expectedType, found.ValueType)
		} else {
			t.Logf("OK: %s -> value_type = '%s'", param.Name, found.ValueType)
		}
	}
}
