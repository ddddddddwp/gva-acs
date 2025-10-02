package core

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/interfaces"
	"go.uber.org/zap"
)

// TR069Builder TR069消息构建器实现
type TR069Builder struct {
	logger *zap.Logger
}

// NewTR069Builder 创建TR069构建器
func NewTR069Builder(logger *zap.Logger) *TR069Builder {
	return &TR069Builder{
		logger: logger,
	}
}

// BuildRPCResponse 构建RPC响应消息
func (b *TR069Builder) BuildRPCResponse(ctx context.Context, method string, params map[string]interface{}) (interface{}, error) {
	switch method {
	case "Inform":
		return b.buildInformResponseFromParams(ctx, params)
	case "GetParameterValues":
		return b.buildGetParameterValuesRequest(ctx, params)
	case "SetParameterValues":
		return b.buildSetParameterValuesRequest(ctx, params)
	case "Reboot":
		return b.buildRebootRequest(ctx, params)
	case "FactoryReset":
		return b.buildFactoryResetRequest(ctx, params)
	case "Download":
		return b.buildDownloadRequest(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported RPC method: %s", method)
	}
}

// BuildInformResponse 构建Inform响应
func (b *TR069Builder) BuildInformResponse(ctx context.Context, maxEnvelopes int) (interface{}, error) {
	response := &InformResponseStruct{
		MaxEnvelopes: maxEnvelopes,
	}
	
	b.logger.Debug("Built inform response", 
		zap.Int("max_envelopes", maxEnvelopes))
	
	return response, nil
}

// BuildGetParameterValuesRequest 构建获取参数值请求
func (b *TR069Builder) BuildGetParameterValuesRequest(ctx context.Context, parameterNames []string) (interface{}, error) {
	if len(parameterNames) == 0 {
		return nil, fmt.Errorf("parameter names cannot be empty")
	}
	
	request := &GetParameterValuesRequestStruct{
		ParameterNames: parameterNames,
	}
	
	b.logger.Debug("Built get parameter values request", 
		zap.Strings("parameter_names", parameterNames))
	
	return request, nil
}

// BuildSetParameterValuesRequest 构建设置参数值请求
func (b *TR069Builder) BuildSetParameterValuesRequest(ctx context.Context, parameters []interfaces.Parameter) (interface{}, error) {
	if len(parameters) == 0 {
		return nil, fmt.Errorf("parameters cannot be empty")
	}
	
	parameterList := make([]ParameterValueStruct, len(parameters))
	for i, param := range parameters {
		parameterList[i] = ParameterValueStruct{
			Name:  param.Name,
			Value: param.Value,
			Type:  b.getXSDType(param.Value),
		}
	}
	
	request := &SetParameterValuesRequestStruct{
		ParameterList: parameterList,
		ParameterKey:  fmt.Sprintf("key_%d", time.Now().Unix()),
	}
	
	b.logger.Debug("Built set parameter values request", 
		zap.Int("parameter_count", len(parameters)))
	
	return request, nil
}

// BuildRebootRequest 构建重启请求
func (b *TR069Builder) BuildRebootRequest(ctx context.Context, commandKey string) (interface{}, error) {
	if commandKey == "" {
		commandKey = fmt.Sprintf("reboot_%d", time.Now().Unix())
	}
	
	request := &RebootRequestStruct{
		CommandKey: commandKey,
	}
	
	b.logger.Debug("Built reboot request", 
		zap.String("command_key", commandKey))
	
	return request, nil
}

// BuildFactoryResetRequest 构建恢复出厂设置请求
func (b *TR069Builder) BuildFactoryResetRequest(ctx context.Context, commandKey string) (interface{}, error) {
	if commandKey == "" {
		commandKey = fmt.Sprintf("factory_reset_%d", time.Now().Unix())
	}
	
	request := &FactoryResetRequestStruct{
		CommandKey: commandKey,
	}
	
	b.logger.Debug("Built factory reset request", 
		zap.String("command_key", commandKey))
	
	return request, nil
}

// BuildDownloadRequest 构建下载请求
func (b *TR069Builder) BuildDownloadRequest(ctx context.Context, fileType, url, username, password, commandKey string) (interface{}, error) {
	if fileType == "" {
		return nil, fmt.Errorf("file type cannot be empty")
	}
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}
	if commandKey == "" {
		commandKey = fmt.Sprintf("download_%d", time.Now().Unix())
	}
	
	request := &DownloadRequestStruct{
		CommandKey:     commandKey,
		FileType:       fileType,
		URL:            url,
		Username:       username,
		Password:       password,
		FileSize:       0, // 将在实际下载时确定
		TargetFileName: "",
		DelaySeconds:   0,
		SuccessURL:     "",
		FailureURL:     "",
	}
	
	b.logger.Debug("Built download request", 
		zap.String("command_key", commandKey),
		zap.String("file_type", fileType),
		zap.String("url", url))
	
	return request, nil
}

// buildInformResponseFromParams 从参数构建Inform响应
func (b *TR069Builder) buildInformResponseFromParams(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	maxEnvelopes := 1
	if me, ok := params["MaxEnvelopes"]; ok {
		if meInt, ok := me.(int); ok {
			maxEnvelopes = meInt
		}
	}
	
	return b.BuildInformResponse(ctx, maxEnvelopes)
}

// buildGetParameterValuesRequest 从参数构建获取参数值请求
func (b *TR069Builder) buildGetParameterValuesRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	parameterNames := []string{}
	if pn, ok := params["ParameterNames"]; ok {
		if pnSlice, ok := pn.([]string); ok {
			parameterNames = pnSlice
		} else if pnInterface, ok := pn.([]interface{}); ok {
			for _, name := range pnInterface {
				if nameStr, ok := name.(string); ok {
					parameterNames = append(parameterNames, nameStr)
				}
			}
		}
	}
	
	return b.BuildGetParameterValuesRequest(ctx, parameterNames)
}

// buildSetParameterValuesRequest 从参数构建设置参数值请求
func (b *TR069Builder) buildSetParameterValuesRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var parameters []interfaces.Parameter
	
	if pl, ok := params["ParameterList"]; ok {
		if plSlice, ok := pl.([]interfaces.Parameter); ok {
			parameters = plSlice
		} else if plInterface, ok := pl.([]interface{}); ok {
			for _, param := range plInterface {
				if paramMap, ok := param.(map[string]interface{}); ok {
					parameter := interfaces.Parameter{
						Name:      getString(paramMap, "Name"),
						Value:     paramMap["Value"],
						Type:      getString(paramMap, "Type"),
						Writable:  getBool(paramMap, "Writable"),
						Timestamp: time.Now(),
					}
					parameters = append(parameters, parameter)
				}
			}
		}
	}
	
	return b.BuildSetParameterValuesRequest(ctx, parameters)
}

// buildRebootRequest 从参数构建重启请求
func (b *TR069Builder) buildRebootRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	commandKey := getString(params, "CommandKey")
	return b.BuildRebootRequest(ctx, commandKey)
}

// buildFactoryResetRequest 从参数构建恢复出厂设置请求
func (b *TR069Builder) buildFactoryResetRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	commandKey := getString(params, "CommandKey")
	return b.BuildFactoryResetRequest(ctx, commandKey)
}

// buildDownloadRequest 从参数构建下载请求
func (b *TR069Builder) buildDownloadRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	fileType := getString(params, "FileType")
	url := getString(params, "URL")
	username := getString(params, "Username")
	password := getString(params, "Password")
	commandKey := getString(params, "CommandKey")
	
	return b.BuildDownloadRequest(ctx, fileType, url, username, password, commandKey)
}

// getXSDType 获取XSD类型
func (b *TR069Builder) getXSDType(value interface{}) string {
	switch value.(type) {
	case bool:
		return "xsd:boolean"
	case int, int8, int16, int32, int64:
		return "xsd:int"
	case uint, uint8, uint16, uint32, uint64:
		return "xsd:unsignedInt"
	case float32, float64:
		return "xsd:double"
	case time.Time:
		return "xsd:dateTime"
	default:
		return "xsd:string"
	}
}

// XML结构定义

// InformResponseStruct Inform响应结构
type InformResponseStruct struct {
	XMLName      xml.Name `xml:"cwmp:InformResponse"`
	MaxEnvelopes int      `xml:"MaxEnvelopes"`
}

// GetParameterValuesRequestStruct 获取参数值请求结构
type GetParameterValuesRequestStruct struct {
	XMLName        xml.Name `xml:"cwmp:GetParameterValues"`
	ParameterNames []string `xml:"ParameterNames>string"`
}

// SetParameterValuesRequestStruct 设置参数值请求结构
type SetParameterValuesRequestStruct struct {
	XMLName       xml.Name               `xml:"cwmp:SetParameterValues"`
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
	ParameterKey  string                 `xml:"ParameterKey"`
}

// ParameterValueStruct 参数值结构
type ParameterValueStruct struct {
	Name  string      `xml:"Name"`
	Value interface{} `xml:"Value"`
	Type  string      `xml:"Value>type,attr"`
}

// RebootRequestStruct 重启请求结构
type RebootRequestStruct struct {
	XMLName    xml.Name `xml:"cwmp:Reboot"`
	CommandKey string   `xml:"CommandKey"`
}

// FactoryResetRequestStruct 恢复出厂设置请求结构
type FactoryResetRequestStruct struct {
	XMLName    xml.Name `xml:"cwmp:FactoryReset"`
	CommandKey string   `xml:"CommandKey"`
}

// DownloadRequestStruct 下载请求结构
type DownloadRequestStruct struct {
	XMLName        xml.Name `xml:"cwmp:Download"`
	CommandKey     string   `xml:"CommandKey"`
	FileType       string   `xml:"FileType"`
	URL            string   `xml:"URL"`
	Username       string   `xml:"Username"`
	Password       string   `xml:"Password"`
	FileSize       int64    `xml:"FileSize"`
	TargetFileName string   `xml:"TargetFileName"`
	DelaySeconds   int      `xml:"DelaySeconds"`
	SuccessURL     string   `xml:"SuccessURL"`
	FailureURL     string   `xml:"FailureURL"`
}

// GetParameterValuesResponseStruct 获取参数值响应结构
type GetParameterValuesResponseStruct struct {
	XMLName       xml.Name               `xml:"cwmp:GetParameterValuesResponse"`
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

// SetParameterValuesResponseStruct 设置参数值响应结构
type SetParameterValuesResponseStruct struct {
	XMLName xml.Name `xml:"cwmp:SetParameterValuesResponse"`
	Status  int      `xml:"Status"`
}

// RebootResponseStruct 重启响应结构
type RebootResponseStruct struct {
	XMLName xml.Name `xml:"cwmp:RebootResponse"`
	Status  int      `xml:"Status"`
}

// FactoryResetResponseStruct 恢复出厂设置响应结构
type FactoryResetResponseStruct struct {
	XMLName xml.Name `xml:"cwmp:FactoryResetResponse"`
	Status  int      `xml:"Status"`
}

// DownloadResponseStruct 下载响应结构
type DownloadResponseStruct struct {
	XMLName      xml.Name   `xml:"cwmp:DownloadResponse"`
	Status       int        `xml:"Status"`
	StartTime    time.Time  `xml:"StartTime"`
	CompleteTime *time.Time `xml:"CompleteTime,omitempty"`
}

// TransferCompleteResponseStruct 传输完成响应结构
type TransferCompleteResponseStruct struct {
	XMLName xml.Name `xml:"cwmp:TransferCompleteResponse"`
	Status  int      `xml:"Status"`
}

// 辅助函数

// getString 从map中获取字符串值
func getString(m map[string]interface{}, key string) string {
	if value, ok := m[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getBool 从map中获取布尔值
func getBool(m map[string]interface{}, key string) bool {
	if value, ok := m[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// getInt 从map中获取整数值
func getInt(m map[string]interface{}, key string) int {
	if value, ok := m[key]; ok {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// validateParameters 验证参数列表
func (b *TR069Builder) validateParameters(parameters []interfaces.Parameter) error {
	if len(parameters) == 0 {
		return fmt.Errorf("parameter list cannot be empty")
	}
	
	for i, param := range parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter %d: name cannot be empty", i)
		}
	}
	
	return nil
}

// validateURL 验证URL格式
func (b *TR069Builder) validateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	
	// 简单的URL验证
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "ftp://") {
		return fmt.Errorf("invalid URL format: %s", url)
	}
	
	return nil
}