package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	Method     string
	Permission string
	Transfer   bool
	Validate   func(any) error
}

var RPCSpecs = map[string]RPCSpec{
	"GetRPCMethods":          {Method: "GetRPCMethods", Permission: "query"},
	"GetParameterValues":     {Method: "GetParameterValues", Permission: "query", Validate: validateGPV},
	"GetParameterNames":      {Method: "GetParameterNames", Permission: "query", Validate: validateGPN},
	"GetParameterAttributes": {Method: "GetParameterAttributes", Permission: "query", Validate: validateGPA},
	"SetParameterValues":     {Method: "SetParameterValues", Permission: "config", Validate: validateSPV},
	"SetParameterAttributes": {Method: "SetParameterAttributes", Permission: "config", Validate: validateSPA},
	"AddObject":              {Method: "AddObject", Permission: "config", Validate: validateObject},
	"DeleteObject":           {Method: "DeleteObject", Permission: "config", Validate: validateObject},
	"Download":               {Method: "Download", Permission: "transfer", Transfer: true, Validate: validateDownload},
	"Upload":                 {Method: "Upload", Permission: "transfer", Transfer: true, Validate: validateUpload},
	"Reboot":                 {Method: "Reboot", Permission: "maintenance"},
	"FactoryReset":           {Method: "FactoryReset", Permission: "maintenance"},
}

var (
	objectNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z0-9_-]+)+\.$`)
	allowedXSDTypes   = map[string]struct{}{
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

func ValidateRPCRequest(method string, request any) error {
	spec, ok := RPCSpecs[method]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownRPCMethod, method)
	}
	if spec.Validate == nil {
		return nil
	}
	return spec.Validate(request)
}

func ValidateRPCSubmission(ctx context.Context, db *gorm.DB, deviceID uint, method string, now time.Time) error {
	if _, ok := RPCSpecs[method]; !ok {
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
	if method == "GetRPCMethods" {
		return nil
	}

	var capabilities model.DeviceRPCMethods
	if err := db.WithContext(ctx).Where("device_id = ?", deviceID).First(&capabilities).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRPCCapabilitiesUnknown
		}
		return err
	}

	var methods []string
	if err := json.Unmarshal(capabilities.MethodsJSON, &methods); err != nil {
		return fmt.Errorf("解析设备能力失败: %w", err)
	}
	if len(methods) == 0 {
		return ErrRPCCapabilitiesUnknown
	}
	for _, supported := range methods {
		if supported == method {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrRPCMethodUnsupported, method)
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
