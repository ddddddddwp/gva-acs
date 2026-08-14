package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"gorm.io/gorm"
)

var (
	ErrUnknownRPCMethod       = errors.New("unknown RPC method")
	ErrRPCDeviceNotFound      = errors.New("设备不存在")
	ErrRPCCapabilitiesUnknown = errors.New("请先查询设备能力")
	ErrRPCMethodUnsupported   = errors.New("设备未声明支持该方法")
)

type RPCSpec struct {
	Method           string
	DisplayName      string
	Operation        string
	Capability       string
	Permission       string
	Confirmation     RPCConfirmationPolicy
	ResultPolicy     RPCResultPolicy
	DeferredPolicy   RPCDeferredPolicy
	Transfer         bool
	ServerCommandKey bool
	Validate         func(any) error
	newRequest       func() any
	normalize        func(any) (map[string]interface{}, error)
}

type RPCConfirmationPolicy string

const (
	RPCConfirmationNone   RPCConfirmationPolicy = "none"
	RPCConfirmationNormal RPCConfirmationPolicy = "normal"
	RPCConfirmationDanger RPCConfirmationPolicy = "danger"
)

type RPCResultPolicy string

const (
	RPCResultMethods             RPCResultPolicy = "methods"
	RPCResultParameterValues     RPCResultPolicy = "parameterValues"
	RPCResultParameterInfos      RPCResultPolicy = "parameterInfos"
	RPCResultParameterAttributes RPCResultPolicy = "parameterAttributes"
	RPCResultStatus              RPCResultPolicy = "status"
	RPCResultObjectStatus        RPCResultPolicy = "objectStatus"
	RPCResultTransfer            RPCResultPolicy = "transfer"
	RPCResultAcknowledgement     RPCResultPolicy = "acknowledgement"
)

type RPCDeferredPolicy string

const (
	RPCDeferredNone             RPCDeferredPolicy = "none"
	RPCDeferredTransferComplete RPCDeferredPolicy = "transferComplete"
)

var RPCSpecs = map[string]RPCSpec{
	"GetRPCMethods": {
		Method: "GetRPCMethods", DisplayName: "查询设备能力", Operation: "GetRPCMethods", Permission: "query",
		Confirmation: RPCConfirmationNone, ResultPolicy: RPCResultMethods, DeferredPolicy: RPCDeferredNone,
		newRequest: newRPCRequest[emptyRPCRequest], normalize: normalizeEmptyRequest,
	},
	"GetParameterValues": {
		Method: "GetParameterValues", DisplayName: "获取参数", Operation: "GetParameterValues", Capability: "GetParameterValues", Permission: "query",
		Confirmation: RPCConfirmationNone, ResultPolicy: RPCResultParameterValues, DeferredPolicy: RPCDeferredNone,
		Validate: validateGPV, newRequest: newRPCRequest[req.GetParameterValuesRequest], normalize: normalizeGPV,
	},
	"GetParameterNames": {
		Method: "GetParameterNames", DisplayName: "获取参数名称", Operation: "GetParameterNames", Capability: "GetParameterNames", Permission: "query",
		Confirmation: RPCConfirmationNone, ResultPolicy: RPCResultParameterInfos, DeferredPolicy: RPCDeferredNone,
		Validate: validateGPN, newRequest: newRPCRequest[req.GetParameterNamesRequest], normalize: normalizeGPN,
	},
	"GetParameterAttributes": {
		Method: "GetParameterAttributes", DisplayName: "获取参数属性", Operation: "GetParameterAttributes", Capability: "GetParameterAttributes", Permission: "query",
		Confirmation: RPCConfirmationNone, ResultPolicy: RPCResultParameterAttributes, DeferredPolicy: RPCDeferredNone,
		Validate: validateGPA, newRequest: newRPCRequest[req.GetParameterAttributesRequest], normalize: normalizeGPA,
	},
	"SetParameterValues": {
		Method: "SetParameterValues", DisplayName: "配置参数", Operation: "SetParameterValues", Capability: "SetParameterValues", Permission: "config",
		Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultStatus, DeferredPolicy: RPCDeferredNone,
		Validate: validateSPV, newRequest: newRPCRequest[req.SetParameterValuesRequest], normalize: normalizeSPV,
	},
	"SetParameterAttributes": {
		Method: "SetParameterAttributes", DisplayName: "配置参数属性", Operation: "SetParameterAttributes", Capability: "SetParameterAttributes", Permission: "config",
		Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultStatus, DeferredPolicy: RPCDeferredNone,
		Validate: validateSPA, newRequest: newRPCRequest[req.SetParameterAttributesRequest], normalize: normalizeSPA,
	},
	"AddObject": {
		Method: "AddObject", DisplayName: "添加对象", Operation: "AddObject", Capability: "AddObject", Permission: "config",
		Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultObjectStatus, DeferredPolicy: RPCDeferredNone,
		Validate: validateObject, newRequest: newRPCRequest[req.ObjectRequest], normalize: normalizeObject,
	},
	"DeleteObject": {
		Method: "DeleteObject", DisplayName: "删除对象", Operation: "DeleteObject", Capability: "DeleteObject", Permission: "config",
		Confirmation: RPCConfirmationDanger, ResultPolicy: RPCResultStatus, DeferredPolicy: RPCDeferredNone,
		Validate: validateObjectInstance, newRequest: newRPCRequest[req.ObjectRequest], normalize: normalizeObject,
	},
	"Download": {
		Method: "Download", DisplayName: "下载文件", Operation: "Download", Capability: "Download", Permission: "transfer",
		Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultTransfer, DeferredPolicy: RPCDeferredTransferComplete,
		Transfer: true, ServerCommandKey: true,
		Validate: validateDownload, newRequest: newRPCRequest[req.DownloadRequest], normalize: normalizeDownload,
	},
	"Upload": {
		Method: "Upload", DisplayName: "上传文件", Operation: "Upload", Capability: "Upload", Permission: "transfer",
		Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultTransfer, DeferredPolicy: RPCDeferredTransferComplete,
		Transfer: true, ServerCommandKey: true,
		Validate: validateUpload, newRequest: newRPCRequest[req.UploadRequest], normalize: normalizeUpload,
	},
	"Reboot": {
		Method: "Reboot", DisplayName: "重启设备", Operation: "Reboot", Capability: "Reboot", Permission: "maintenance",
		Confirmation: RPCConfirmationDanger, ResultPolicy: RPCResultAcknowledgement, DeferredPolicy: RPCDeferredNone,
		ServerCommandKey: true,
		newRequest:       newRPCRequest[emptyRPCRequest],
		normalize:        normalizeEmptyRequest,
	},
	"FactoryReset": {
		Method: "FactoryReset", DisplayName: "恢复出厂设置", Operation: "FactoryReset", Capability: "FactoryReset", Permission: "maintenance",
		Confirmation: RPCConfirmationDanger, ResultPolicy: RPCResultAcknowledgement, DeferredPolicy: RPCDeferredNone,
		newRequest: newRPCRequest[emptyRPCRequest], normalize: normalizeEmptyRequest,
	},
}

var (
	objectNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z0-9_-]+)+\.$`)
	objectInstancePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z0-9_-]+)*\.[0-9]+\.$`)
	allowedXSDTypes       = map[string]struct{}{
		"xsd:string":       {},
		"xsd:int":          {},
		"xsd:unsignedInt":  {},
		"xsd:boolean":      {},
		"xsd:dateTime":     {},
		"xsd:base64Binary": {},
		"xsd:long":         {},
		"xsd:unsignedLong": {},
		"xsd:double":       {},
	}
)

type emptyRPCRequest struct{}

func ValidateRPCRequest(method string, request any) error {
	spec, ok := RPCSpecs[method]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	if err := validateRPCRequestType(spec, request); err != nil {
		return err
	}
	if spec.Validate == nil {
		return nil
	}
	return spec.Validate(request)
}

func validateRPCRequestType(spec RPCSpec, request any) error {
	if request == nil && spec.Validate == nil {
		return nil
	}
	expected := reflect.TypeOf(spec.newRequest()).Elem()
	actual := reflect.TypeOf(request)
	if actual != nil && actual.Kind() == reflect.Pointer {
		if reflect.ValueOf(request).IsNil() {
			actual = nil
		} else {
			actual = actual.Elem()
		}
	}
	if actual != expected {
		return fmt.Errorf("invalid request type %T, want %s", request, expected)
	}
	return nil
}

func EncodeRPCRequest(method string, request any) ([]byte, error) {
	spec, ok := RPCSpecs[method]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	if request == nil && spec.Validate == nil {
		request = reflect.ValueOf(spec.newRequest()).Elem().Interface()
	}
	if err := ValidateRPCRequest(method, request); err != nil {
		return nil, err
	}
	persisted, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode %s request: %w", method, err)
	}
	return persisted, nil
}

func DecodeRPCRequest(method string, persisted []byte) (any, error) {
	spec, ok := RPCSpecs[method]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	target := spec.newRequest()
	if err := json.Unmarshal(persisted, target); err != nil {
		return nil, fmt.Errorf("decode %s request: %w", method, err)
	}
	request := reflect.ValueOf(target).Elem().Interface()
	if err := ValidateRPCRequest(method, request); err != nil {
		return nil, err
	}
	return request, nil
}

func NormalizeRPCRequest(method string, request any) (map[string]interface{}, error) {
	spec, ok := RPCSpecs[method]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	if err := ValidateRPCRequest(method, request); err != nil {
		return nil, err
	}
	return spec.normalize(request)
}

func DecodeRPCParams(method string, persisted []byte) (map[string]interface{}, error) {
	request, err := DecodeRPCRequest(method, persisted)
	if err != nil {
		return nil, err
	}
	return NormalizeRPCRequest(method, request)
}

func newRPCRequest[T any]() any {
	return new(T)
}

func normalizeEmptyRequest(any) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func normalizeGPV(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.GetParameterValuesRequest](input)
	if !ok {
		return nil, invalidRequestType[req.GetParameterValuesRequest](input)
	}
	return map[string]interface{}{"paths": append([]string(nil), in.Paths...)}, nil
}

func normalizeGPN(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.GetParameterNamesRequest](input)
	if !ok {
		return nil, invalidRequestType[req.GetParameterNamesRequest](input)
	}
	return map[string]interface{}{"parameterPath": in.ParameterPath, "nextLevel": in.NextLevel}, nil
}

func normalizeGPA(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.GetParameterAttributesRequest](input)
	if !ok {
		return nil, invalidRequestType[req.GetParameterAttributesRequest](input)
	}
	return map[string]interface{}{"parameterNames": append([]string(nil), in.ParameterNames...)}, nil
}

func normalizeSPV(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.SetParameterValuesRequest](input)
	if !ok {
		return nil, invalidRequestType[req.SetParameterValuesRequest](input)
	}
	parameters := make([]map[string]interface{}, 0, len(in.Parameters))
	for _, parameter := range in.Parameters {
		parameters = append(parameters, map[string]interface{}{
			"name": parameter.Name, "type": parameter.Type, "value": parameter.Value,
		})
	}
	return map[string]interface{}{"parameterKey": in.ParameterKey, "parameters": parameters}, nil
}

func normalizeSPA(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.SetParameterAttributesRequest](input)
	if !ok {
		return nil, invalidRequestType[req.SetParameterAttributesRequest](input)
	}
	attributes := make([]map[string]interface{}, 0, len(in.ParameterAttributes))
	for _, attribute := range in.ParameterAttributes {
		attributes = append(attributes, map[string]interface{}{
			"name":               attribute.Name,
			"notificationChange": attribute.NotificationChange,
			"notification":       attribute.Notification,
			"accessListChange":   attribute.AccessListChange,
			"accessList":         append([]string(nil), attribute.AccessList...),
		})
	}
	return map[string]interface{}{"parameterAttributes": attributes}, nil
}

func normalizeObject(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.ObjectRequest](input)
	if !ok {
		return nil, invalidRequestType[req.ObjectRequest](input)
	}
	return map[string]interface{}{"objectName": in.ObjectName, "parameterKey": in.ParameterKey}, nil
}

func normalizeDownload(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.DownloadRequest](input)
	if !ok {
		return nil, invalidRequestType[req.DownloadRequest](input)
	}
	return map[string]interface{}{
		"fileType": in.FileType, "url": in.URL, "username": in.Username, "password": in.Password,
		"fileSize": in.FileSize, "targetFileName": in.TargetFileName, "delaySeconds": in.DelaySeconds,
		"successURL": in.SuccessURL, "failureURL": in.FailureURL,
	}, nil
}

func normalizeUpload(input any) (map[string]interface{}, error) {
	in, ok := requestValue[req.UploadRequest](input)
	if !ok {
		return nil, invalidRequestType[req.UploadRequest](input)
	}
	return map[string]interface{}{
		"fileType": in.FileType, "url": in.URL, "username": in.Username, "password": in.Password,
		"delaySeconds": in.DelaySeconds,
	}, nil
}

func ValidateRPCSubmission(ctx context.Context, db *gorm.DB, deviceID uint, method string, now time.Time) error {
	spec, ok := RPCSpecs[method]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	if db == nil {
		return errors.New("db not initialized")
	}

	var device model.Device
	if err := db.WithContext(ctx).Select("id", "last_inform").First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: %d", ErrRPCDeviceNotFound, deviceID)
		}
		return err
	}
	if device.LastInform.IsZero() || now.Sub(device.LastInform) >= commandOnlineThreshold {
		return ErrDeviceOffline
	}
	if spec.Capability == "" {
		return nil
	}

	var capabilities model.DeviceRPCMethods
	if err := db.WithContext(ctx).Where("device_id = ?", deviceID).First(&capabilities).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRPCCapabilitiesUnknown
		}
		return err
	}

	methods, err := decodePersistedRPCMethods(capabilities.MethodsJSON)
	if err != nil {
		return err
	}
	for _, supported := range methods {
		if supported == spec.Capability {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrRPCMethodUnsupported, method)
}

func decodePersistedRPCMethods(raw []byte) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, ErrRPCCapabilitiesUnknown
	}
	var methods []string
	if err := json.Unmarshal(trimmed, &methods); err != nil {
		return nil, fmt.Errorf("解析设备能力失败: %w", err)
	}
	if len(methods) == 0 {
		return nil, ErrRPCCapabilitiesUnknown
	}
	return methods, nil
}

func validateGPV(input any) error {
	in, ok := requestValue[req.GetParameterValuesRequest](input)
	if !ok {
		return invalidRequestType[req.GetParameterValuesRequest](input)
	}
	return validateNonEmptyStrings("paths", in.Paths)
}

func validateGPN(input any) error {
	in, ok := requestValue[req.GetParameterNamesRequest](input)
	if !ok {
		return invalidRequestType[req.GetParameterNamesRequest](input)
	}
	if strings.TrimSpace(in.ParameterPath) == "" {
		return errors.New("parameterPath is required")
	}
	return nil
}

func validateGPA(input any) error {
	in, ok := requestValue[req.GetParameterAttributesRequest](input)
	if !ok {
		return invalidRequestType[req.GetParameterAttributesRequest](input)
	}
	return validateNonEmptyStrings("parameterNames", in.ParameterNames)
}

func validateSPV(input any) error {
	in, ok := requestValue[req.SetParameterValuesRequest](input)
	if !ok {
		return invalidRequestType[req.SetParameterValuesRequest](input)
	}
	if len(in.Parameters) == 0 {
		return errors.New("parameters is required")
	}
	for index, parameter := range in.Parameters {
		if strings.TrimSpace(parameter.Name) == "" {
			return fmt.Errorf("parameters[%d].name is required", index)
		}
		if _, ok := allowedXSDTypes[parameter.Type]; !ok {
			return fmt.Errorf("parameters[%d].type %q is not supported", index, parameter.Type)
		}
		if parameter.Value == nil {
			return fmt.Errorf("parameters[%d].value is required", index)
		}
	}
	return nil
}

func validateSPA(input any) error {
	in, ok := requestValue[req.SetParameterAttributesRequest](input)
	if !ok {
		return invalidRequestType[req.SetParameterAttributesRequest](input)
	}
	if len(in.ParameterAttributes) == 0 {
		return errors.New("parameterAttributes is required")
	}
	for index, attribute := range in.ParameterAttributes {
		if strings.TrimSpace(attribute.Name) == "" {
			return fmt.Errorf("parameterAttributes[%d].name is required", index)
		}
		if attribute.Notification < 0 || attribute.Notification > 2 {
			return fmt.Errorf("parameterAttributes[%d].notification must be between 0 and 2", index)
		}
		if attribute.AccessListChange {
			for accessIndex, entry := range attribute.AccessList {
				if strings.TrimSpace(entry) == "" {
					return fmt.Errorf("parameterAttributes[%d].accessList[%d] is required", index, accessIndex)
				}
			}
		}
	}
	return nil
}

func validateObject(input any) error {
	in, ok := requestValue[req.ObjectRequest](input)
	if !ok {
		return invalidRequestType[req.ObjectRequest](input)
	}
	if !objectNamePattern.MatchString(in.ObjectName) {
		return errors.New("objectName must be a dot-terminated TR-069 object path")
	}
	return nil
}

func validateObjectInstance(input any) error {
	in, ok := requestValue[req.ObjectRequest](input)
	if !ok {
		return invalidRequestType[req.ObjectRequest](input)
	}
	if !objectInstancePattern.MatchString(in.ObjectName) {
		return errors.New("objectName must be a complete dot-terminated numeric instance path")
	}
	return nil
}

func validateDownload(input any) error {
	in, ok := requestValue[req.DownloadRequest](input)
	if !ok {
		return invalidRequestType[req.DownloadRequest](input)
	}
	if err := validateTransfer("download", in.FileType, in.URL, in.DelaySeconds); err != nil {
		return err
	}
	if in.FileSize < 0 {
		return errors.New("download.fileSize must not be negative")
	}
	if err := validateOptionalTransferURL("download.successURL", in.SuccessURL); err != nil {
		return err
	}
	return validateOptionalTransferURL("download.failureURL", in.FailureURL)
}

func validateUpload(input any) error {
	in, ok := requestValue[req.UploadRequest](input)
	if !ok {
		return invalidRequestType[req.UploadRequest](input)
	}
	return validateTransfer("upload", in.FileType, in.URL, in.DelaySeconds)
}

func validateTransfer(name, fileType, rawURL string, delaySeconds int) error {
	if strings.TrimSpace(fileType) == "" {
		return fmt.Errorf("%s.fileType is required", name)
	}
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("%s.url is required", name)
	}
	if err := validateTransferURL(name+".url", rawURL); err != nil {
		return err
	}
	if delaySeconds < 0 {
		return fmt.Errorf("%s.delaySeconds must not be negative", name)
	}
	return nil
}

func validateOptionalTransferURL(field, rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return nil
	}
	return validateTransferURL(field, rawURL)
}

func validateTransferURL(field, rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute HTTP, HTTPS, or FTP URL", field)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "ftp":
		return nil
	default:
		return fmt.Errorf("%s must be an absolute HTTP, HTTPS, or FTP URL", field)
	}
}

func validateNonEmptyStrings(field string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s is required", field)
	}
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%d] is required", field, index)
		}
	}
	return nil
}

func requestValue[T any](input any) (T, bool) {
	if value, ok := input.(T); ok {
		return value, true
	}
	if pointer, ok := input.(*T); ok && pointer != nil {
		return *pointer, true
	}
	var zero T
	return zero, false
}

func invalidRequestType[T any](input any) error {
	var expected T
	return fmt.Errorf("invalid request type %T, want %T", input, expected)
}
