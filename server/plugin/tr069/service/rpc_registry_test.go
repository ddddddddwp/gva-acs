package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRPCSpecsContainExactMethodMetadata(t *testing.T) {
	want := map[string]struct {
		displayName  string
		operation    string
		capability   string
		permission   string
		confirmation string
		resultPolicy string
		deferred     string
		transfer     bool
	}{
		"GetRPCMethods":          {displayName: "查询设备能力", operation: "GetRPCMethods", permission: "query", confirmation: "none", resultPolicy: "methods", deferred: "none"},
		"GetParameterValues":     {displayName: "获取参数", operation: "GetParameterValues", capability: "GetParameterValues", permission: "query", confirmation: "none", resultPolicy: "parameterValues", deferred: "none"},
		"GetParameterNames":      {displayName: "获取参数名称", operation: "GetParameterNames", capability: "GetParameterNames", permission: "query", confirmation: "none", resultPolicy: "parameterInfos", deferred: "none"},
		"GetParameterAttributes": {displayName: "获取参数属性", operation: "GetParameterAttributes", capability: "GetParameterAttributes", permission: "query", confirmation: "none", resultPolicy: "parameterAttributes", deferred: "none"},
		"SetParameterValues":     {displayName: "配置参数", operation: "SetParameterValues", capability: "SetParameterValues", permission: "config", confirmation: "normal", resultPolicy: "status", deferred: "none"},
		"SetParameterAttributes": {displayName: "配置参数属性", operation: "SetParameterAttributes", capability: "SetParameterAttributes", permission: "config", confirmation: "normal", resultPolicy: "status", deferred: "none"},
		"AddObject":              {displayName: "添加对象", operation: "AddObject", capability: "AddObject", permission: "config", confirmation: "normal", resultPolicy: "objectStatus", deferred: "none"},
		"DeleteObject":           {displayName: "删除对象", operation: "DeleteObject", capability: "DeleteObject", permission: "config", confirmation: "danger", resultPolicy: "status", deferred: "none"},
		"Download":               {displayName: "下载文件", operation: "Download", capability: "Download", permission: "transfer", confirmation: "normal", resultPolicy: "transfer", deferred: "transferComplete", transfer: true},
		"Upload":                 {displayName: "上传文件", operation: "Upload", capability: "Upload", permission: "transfer", confirmation: "normal", resultPolicy: "transfer", deferred: "transferComplete", transfer: true},
		"Reboot":                 {displayName: "重启设备", operation: "Reboot", capability: "Reboot", permission: "maintenance", confirmation: "danger", resultPolicy: "acknowledgement", deferred: "none"},
		"FactoryReset":           {displayName: "恢复出厂设置", operation: "FactoryReset", capability: "FactoryReset", permission: "maintenance", confirmation: "danger", resultPolicy: "acknowledgement", deferred: "none"},
	}

	if len(RPCSpecs) != len(want) {
		t.Fatalf("RPCSpecs has %d methods, want %d", len(RPCSpecs), len(want))
	}
	for method, expected := range want {
		spec, ok := RPCSpecs[method]
		if !ok {
			t.Errorf("RPCSpecs missing %q", method)
			continue
		}
		if spec.Method != method ||
			spec.DisplayName != expected.displayName ||
			spec.Operation != expected.operation ||
			spec.Capability != expected.capability ||
			spec.Permission != expected.permission ||
			string(spec.Confirmation) != expected.confirmation ||
			string(spec.ResultPolicy) != expected.resultPolicy ||
			string(spec.DeferredPolicy) != expected.deferred ||
			spec.Transfer != expected.transfer {
			t.Errorf("RPCSpecs[%q] = %#v, want %#v", method, spec, expected)
		}
		if spec.newRequest == nil || spec.normalize == nil {
			t.Errorf("RPCSpecs[%q] missing typed decode/normalize strategy", method)
		}
	}
}

func TestRPCRequestPersistenceRoundTripUsesTypedNormalization(t *testing.T) {
	tests := []struct {
		method  string
		request any
		assert  func(*testing.T, map[string]interface{})
	}{
		{method: "GetParameterValues", request: req.GetParameterValuesRequest{Paths: []string{"Device.", "Device.WiFi."}}, assert: func(t *testing.T, params map[string]interface{}) {
			t.Helper()
			if _, ok := params["paths"].([]string); !ok {
				t.Fatalf("paths type = %T, want []string", params["paths"])
			}
		}},
		{method: "GetParameterNames", request: req.GetParameterNamesRequest{ParameterPath: "Device.WiFi.", NextLevel: true}},
		{method: "GetParameterAttributes", request: req.GetParameterAttributesRequest{ParameterNames: []string{"Device.WiFi.SSID.1.SSID"}}, assert: func(t *testing.T, params map[string]interface{}) {
			t.Helper()
			names, ok := params["parameterNames"].([]string)
			if !ok {
				t.Fatalf("parameterNames type = %T, want []string", params["parameterNames"])
			}
			if want := []string{"Device.WiFi.SSID.1.SSID"}; !reflect.DeepEqual(names, want) {
				t.Fatalf("parameterNames = %#v, want %#v", names, want)
			}
		}},
		{method: "SetParameterValues", request: req.SetParameterValuesRequest{ParameterKey: "set-1", Parameters: []req.SetParameterValue{{Name: "Device.WiFi.SSID.1.SSID", Type: "xsd:string", Value: "lab"}}}, assert: func(t *testing.T, params map[string]interface{}) {
			t.Helper()
			if _, ok := params["parameters"].([]map[string]interface{}); !ok {
				t.Fatalf("parameters type = %T, want []map[string]interface{}", params["parameters"])
			}
		}},
		{method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{ParameterAttributes: []req.SetParameterAttribute{{Name: "Device.WiFi.SSID.1.SSID", NotificationChange: true, Notification: 2}}}, assert: func(t *testing.T, params map[string]interface{}) {
			t.Helper()
			if _, ok := params["parameterAttributes"].([]map[string]interface{}); !ok {
				t.Fatalf("parameterAttributes type = %T, want []map[string]interface{}", params["parameterAttributes"])
			}
		}},
		{method: "AddObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID.", ParameterKey: "add-1"}},
		{method: "DeleteObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID.7.", ParameterKey: "delete-1"}},
		{method: "Download", request: req.DownloadRequest{FileType: "1 Firmware Upgrade Image", URL: "https://acs.example.test/fw.bin", FileSize: 1024, TargetFileName: "fw.bin"}},
		{method: "Upload", request: req.UploadRequest{FileType: "1 Vendor Configuration File", URL: "https://acs.example.test/config.xml"}},
		{method: "Reboot", request: emptyRPCRequest{}},
	}

	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			persisted, err := EncodeRPCRequest(tc.method, tc.request)
			if err != nil {
				t.Fatalf("EncodeRPCRequest() error = %v", err)
			}
			decoded, err := DecodeRPCRequest(tc.method, persisted)
			if err != nil {
				t.Fatalf("DecodeRPCRequest() error = %v", err)
			}
			if reflect.TypeOf(decoded) != reflect.TypeOf(tc.request) {
				t.Fatalf("decoded type = %T, want %T", decoded, tc.request)
			}
			if !reflect.DeepEqual(decoded, tc.request) {
				t.Fatalf("decoded = %#v, want %#v", decoded, tc.request)
			}

			params, err := DecodeRPCParams(tc.method, persisted)
			if err != nil {
				t.Fatalf("DecodeRPCParams() error = %v", err)
			}
			if tc.assert != nil {
				tc.assert(t, params)
			}
		})
	}
}

func TestRPCRequestValidation(t *testing.T) {
	allowedTypes := []string{
		"xsd:string",
		"xsd:int",
		"xsd:unsignedInt",
		"xsd:boolean",
		"xsd:dateTime",
		"xsd:base64Binary",
		"xsd:long",
		"xsd:unsignedLong",
		"xsd:double",
	}
	for _, valueType := range allowedTypes {
		t.Run("SetParameterValues allows "+valueType, func(t *testing.T) {
			in := req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{{
				Name:  "Device.DeviceInfo.Manufacturer",
				Type:  valueType,
				Value: "value",
			}}}
			if err := ValidateRPCRequest("SetParameterValues", in); err != nil {
				t.Fatalf("ValidateRPCRequest() error = %v", err)
			}
		})
	}

	validDownload := req.DownloadRequest{
		FileType:       "1 Firmware Upgrade Image",
		URL:            "https://acs.example.test/firmware.bin",
		FileSize:       1024,
		TargetFileName: "firmware.bin",
	}
	validUpload := req.UploadRequest{
		FileType: "1 Vendor Configuration File",
		URL:      "ftp://acs.example.test/config.xml",
	}

	tests := []struct {
		name    string
		method  string
		request any
		wantErr bool
	}{
		{name: "GetRPCMethods has no body", method: "GetRPCMethods"},
		{name: "GetParameterValues", method: "GetParameterValues", request: req.GetParameterValuesRequest{Paths: []string{"Device."}}},
		{name: "GetParameterValues missing paths", method: "GetParameterValues", request: req.GetParameterValuesRequest{}, wantErr: true},
		{name: "GetParameterValues blank path", method: "GetParameterValues", request: req.GetParameterValuesRequest{Paths: []string{"  "}}, wantErr: true},
		{name: "GetParameterNames", method: "GetParameterNames", request: req.GetParameterNamesRequest{ParameterPath: "Device."}},
		{name: "GetParameterNames missing path", method: "GetParameterNames", request: req.GetParameterNamesRequest{}, wantErr: true},
		{name: "GetParameterAttributes", method: "GetParameterAttributes", request: req.GetParameterAttributesRequest{ParameterNames: []string{"Device.DeviceInfo.Manufacturer"}}},
		{name: "GetParameterAttributes blank name", method: "GetParameterAttributes", request: req.GetParameterAttributesRequest{ParameterNames: []string{""}}, wantErr: true},
		{name: "SetParameterValues missing parameters", method: "SetParameterValues", request: req.SetParameterValuesRequest{}, wantErr: true},
		{name: "SetParameterValues missing name", method: "SetParameterValues", request: req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{{Type: "xsd:string"}}}, wantErr: true},
		{name: "SetParameterValues missing type", method: "SetParameterValues", request: req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{{Name: "Device.DeviceInfo.Manufacturer"}}}, wantErr: true},
		{name: "SetParameterValues rejects unknown XSD type", method: "SetParameterValues", request: req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{{Name: "Device.DeviceInfo.Manufacturer", Type: "string"}}}, wantErr: true},
		{name: "SetParameterAttributes", method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{ParameterAttributes: []req.SetParameterAttribute{{Name: "Device.DeviceInfo.Manufacturer", Notification: 2}}}},
		{name: "SetParameterAttributes missing entries", method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{}, wantErr: true},
		{name: "SetParameterAttributes missing name", method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{ParameterAttributes: []req.SetParameterAttribute{{Notification: 1}}}, wantErr: true},
		{name: "SetParameterAttributes notification below range", method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{ParameterAttributes: []req.SetParameterAttribute{{Name: "Device.DeviceInfo.Manufacturer", Notification: -1}}}, wantErr: true},
		{name: "SetParameterAttributes notification above range", method: "SetParameterAttributes", request: req.SetParameterAttributesRequest{ParameterAttributes: []req.SetParameterAttribute{{Name: "Device.DeviceInfo.Manufacturer", Notification: 3}}}, wantErr: true},
		{name: "AddObject", method: "AddObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID."}},
		{name: "DeleteObject instance", method: "DeleteObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID.7."}},
		{name: "DeleteObject rejects collection", method: "DeleteObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID."}, wantErr: true},
		{name: "DeleteObject rejects nonnumeric instance", method: "DeleteObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID.seven."}, wantErr: true},
		{name: "Object missing trailing dot", method: "AddObject", request: req.ObjectRequest{ObjectName: "Device.WiFi.SSID"}, wantErr: true},
		{name: "Object missing root", method: "AddObject", request: req.ObjectRequest{ObjectName: ".Device.WiFi."}, wantErr: true},
		{name: "Object empty segment", method: "DeleteObject", request: req.ObjectRequest{ObjectName: "Device..WiFi."}, wantErr: true},
		{name: "Download", method: "Download", request: validDownload},
		{name: "Download missing file type", method: "Download", request: func() req.DownloadRequest { in := validDownload; in.FileType = ""; return in }(), wantErr: true},
		{name: "Download missing URL", method: "Download", request: func() req.DownloadRequest { in := validDownload; in.URL = ""; return in }(), wantErr: true},
		{name: "Download invalid URL", method: "Download", request: func() req.DownloadRequest { in := validDownload; in.URL = "://bad"; return in }(), wantErr: true},
		{name: "Download negative file size", method: "Download", request: func() req.DownloadRequest { in := validDownload; in.FileSize = -1; return in }(), wantErr: true},
		{name: "Download negative delay", method: "Download", request: func() req.DownloadRequest { in := validDownload; in.DelaySeconds = -1; return in }(), wantErr: true},
		{name: "Upload", method: "Upload", request: validUpload},
		{name: "Upload missing file type", method: "Upload", request: req.UploadRequest{URL: validUpload.URL}, wantErr: true},
		{name: "Upload invalid URL", method: "Upload", request: req.UploadRequest{FileType: validUpload.FileType, URL: "not a URL"}, wantErr: true},
		{name: "Upload negative delay", method: "Upload", request: req.UploadRequest{FileType: validUpload.FileType, URL: validUpload.URL, DelaySeconds: -1}, wantErr: true},
		{name: "Reboot has no body", method: "Reboot"},
		{name: "FactoryReset has no body", method: "FactoryReset"},
		{name: "unknown method", method: "VendorMethod", wantErr: true},
		{name: "wrong request type", method: "GetParameterValues", request: req.ObjectRequest{}, wantErr: true},
		{name: "Reboot rejects wrong request type", method: "Reboot", request: req.ObjectRequest{}, wantErr: true},
		{name: "GetRPCMethods rejects a request body", method: "GetRPCMethods", request: req.ObjectRequest{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRPCRequest(tt.method, tt.request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateRPCRequest(%q, %#v) error = %v, wantErr %t", tt.method, tt.request, err, tt.wantErr)
			}
		})
	}
}

func TestRebootUsesEmptyRequestAndServerCommandKeyMetadata(t *testing.T) {
	spec := RPCSpecs["Reboot"]
	if !spec.ServerCommandKey {
		t.Fatal("Reboot must use a server-generated CommandKey")
	}
	persisted, err := EncodeRPCRequest("Reboot", nil)
	if err != nil {
		t.Fatalf("EncodeRPCRequest(Reboot): %v", err)
	}
	if string(persisted) != `{}` {
		t.Fatalf("persisted Reboot request = %s, want {}", persisted)
	}
	params, err := DecodeRPCParams("Reboot", persisted)
	if err != nil {
		t.Fatalf("DecodeRPCParams(Reboot): %v", err)
	}
	if len(params) != 0 {
		t.Fatalf("Reboot user params = %#v, want empty", params)
	}
}

func TestDecodePersistedRPCMethodsTreatsEmptyDataAsUnknown(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "nil", raw: nil},
		{name: "zero length", raw: []byte{}},
		{name: "whitespace", raw: []byte("  ")},
		{name: "JSON null", raw: []byte("null")},
		{name: "empty array", raw: []byte("[]")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodePersistedRPCMethods(tt.raw)
			if !errors.Is(err, ErrRPCCapabilitiesUnknown) || !strings.Contains(err.Error(), "请先查询设备能力") {
				t.Fatalf("decodePersistedRPCMethods(%q) error = %v, want ErrRPCCapabilitiesUnknown", tt.raw, err)
			}
		})
	}
}

func TestRequestDTOBindingTags(t *testing.T) {
	tests := []struct {
		typeOf reflect.Type
		field  string
		want   string
	}{
		{typeOf: reflect.TypeOf(req.GetParameterValuesRequest{}), field: "Paths", want: "required,min=1,dive,required"},
		{typeOf: reflect.TypeOf(req.GetParameterNamesRequest{}), field: "ParameterPath", want: "required"},
		{typeOf: reflect.TypeOf(req.GetParameterAttributesRequest{}), field: "ParameterNames", want: "required,min=1,dive,required"},
		{typeOf: reflect.TypeOf(req.SetParameterValuesRequest{}), field: "Parameters", want: "required,min=1"},
		{typeOf: reflect.TypeOf(req.SetParameterAttributesRequest{}), field: "ParameterAttributes", want: "required,min=1"},
		{typeOf: reflect.TypeOf(req.ObjectRequest{}), field: "ObjectName", want: "required"},
	}
	for _, tt := range tests {
		t.Run(tt.typeOf.Name()+"."+tt.field, func(t *testing.T) {
			field, ok := tt.typeOf.FieldByName(tt.field)
			if !ok {
				t.Fatalf("missing field %s", tt.field)
			}
			if got := field.Tag.Get("binding"); got != tt.want {
				t.Fatalf("binding tag = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateRPCSubmissionPolicy(t *testing.T) {
	db := openRPCRegistryTestDB(t)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	missingCapabilities := model.Device{SerialNumber: "NO-CAPS", OUI: "8CE468", LastInform: now.Add(-time.Minute)}
	emptyCapabilities := model.Device{SerialNumber: "EMPTY-CAPS", OUI: "8CE468", LastInform: now.Add(-time.Minute)}
	supported := model.Device{SerialNumber: "SUPPORTED", OUI: "8CE468", LastInform: now.Add(-time.Minute)}
	unsupported := model.Device{SerialNumber: "UNSUPPORTED", OUI: "8CE468", LastInform: now.Add(-time.Minute)}
	offline := model.Device{SerialNumber: "OFFLINE", OUI: "8CE468", LastInform: now.Add(-commandOnlineThreshold)}
	for _, device := range []*model.Device{&missingCapabilities, &emptyCapabilities, &supported, &unsupported, &offline} {
		if err := db.Create(device).Error; err != nil {
			t.Fatalf("create device: %v", err)
		}
	}

	rows := []model.DeviceRPCMethods{
		{DeviceID: emptyCapabilities.ID, MethodsJSON: datatypes.JSON(`[]`)},
		{DeviceID: supported.ID, MethodsJSON: datatypes.JSON(`["GetParameterValues","Reboot"]`)},
		{DeviceID: unsupported.ID, MethodsJSON: datatypes.JSON(`["Reboot"]`)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create capability rows: %v", err)
	}

	tests := []struct {
		name       string
		deviceID   uint
		method     string
		wantError  error
		wantPhrase string
	}{
		{name: "device missing", deviceID: 999999, method: "GetRPCMethods", wantError: ErrRPCDeviceNotFound},
		{name: "offline", deviceID: offline.ID, method: "GetRPCMethods", wantError: ErrDeviceOffline},
		{name: "GetRPCMethods without capability row", deviceID: missingCapabilities.ID, method: "GetRPCMethods"},
		{name: "GetRPCMethods with empty capabilities", deviceID: emptyCapabilities.ID, method: "GetRPCMethods"},
		{name: "missing capability row", deviceID: missingCapabilities.ID, method: "GetParameterValues", wantError: ErrRPCCapabilitiesUnknown, wantPhrase: "请先查询设备能力"},
		{name: "empty capability list", deviceID: emptyCapabilities.ID, method: "GetParameterValues", wantError: ErrRPCCapabilitiesUnknown, wantPhrase: "请先查询设备能力"},
		{name: "supported method", deviceID: supported.ID, method: "GetParameterValues"},
		{name: "unsupported method", deviceID: unsupported.ID, method: "GetParameterValues", wantError: ErrRPCMethodUnsupported},
		{name: "unknown registry method", deviceID: supported.ID, method: "VendorMethod", wantError: ErrUnknownRPCMethod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRPCSubmission(context.Background(), db, tt.deviceID, tt.method, now)
			if tt.wantError == nil {
				if err != nil {
					t.Fatalf("ValidateRPCSubmission() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("ValidateRPCSubmission() error = %v, want errors.Is(%v)", err, tt.wantError)
			}
			if tt.wantPhrase != "" && !strings.Contains(err.Error(), tt.wantPhrase) {
				t.Fatalf("ValidateRPCSubmission() error = %q, want phrase %q", err, tt.wantPhrase)
			}
		})
	}
}

func TestCommandServiceAppliesRPCPolicyBeforeEnqueue(t *testing.T) {
	db := openRPCRegistryTestDB(t)
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	now := time.Now()
	withoutCapabilities := model.Device{SerialNumber: "NO-CAPS-SUBMIT", OUI: "8CE468", LastInform: now}
	withCapabilities := model.Device{SerialNumber: "WITH-CAPS-SUBMIT", OUI: "8CE468", LastInform: now}
	if err := db.Create(&withoutCapabilities).Error; err != nil {
		t.Fatalf("create no-capability device: %v", err)
	}
	if err := db.Create(&withCapabilities).Error; err != nil {
		t.Fatalf("create capability device: %v", err)
	}
	if err := db.Create(&model.DeviceRPCMethods{
		DeviceID:    withCapabilities.ID,
		MethodsJSON: datatypes.JSON(`["GetParameterValues"]`),
	}).Error; err != nil {
		t.Fatalf("create capabilities: %v", err)
	}

	service := new(CommandService)
	_, err := service.EnqueueGetParameterValues(withoutCapabilities.ID, req.GetParameterValuesRequest{Paths: []string{"Device."}})
	if !errors.Is(err, ErrRPCCapabilitiesUnknown) {
		t.Fatalf("missing capabilities error = %v, want ErrRPCCapabilitiesUnknown", err)
	}

	_, err = service.EnqueueGetParameterValues(withCapabilities.ID, req.GetParameterValuesRequest{Paths: []string{" "}})
	if err == nil || errors.Is(err, ErrCommandQueueUnavailable) || !strings.Contains(err.Error(), "paths") {
		t.Fatalf("blank path error = %v, want request validation before queue access", err)
	}
}

func openRPCRegistryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
