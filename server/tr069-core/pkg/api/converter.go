package api

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// TR069Converter 实现TR069协议与HTTP/JSON的转换
type TR069Converter struct {
parser  interfaces.Parser
builder interfaces.Builder
options map[string]interface{}
}

// NewTR069Converter 创建新的转换器
func NewTR069Converter(parser interfaces.Parser, builder interfaces.Builder) *TR069Converter {
return &TR069Converter{
parser:  parser,
builder: builder,
options: make(map[string]interface{}),
}
}

// WithOption 设置转换选项
func (c *TR069Converter) WithOption(key string, value interface{}) *TR069Converter {
c.options[key] = value
return c
}

// JSONToTR069 将JSON格式转换为TR069 XML格式
func (c *TR069Converter) JSONToTR069(jsonData []byte) ([]byte, error) {
// 解析JSON数据
var jsonObj map[string]interface{}
if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
return nil, fmt.Errorf("failed to parse JSON: %w", err)
}

// 提取RPC方法和参数
method, ok := jsonObj["method"].(string)
if !ok {
return nil, errors.New("missing or invalid 'method' field in JSON")
}

params, ok := jsonObj["params"].(map[string]interface{})
if !ok {
params = make(map[string]interface{})
}

// 构建TR069消息
return c.builder.BuildRPCRequest(context.Background(), method, params)
}

// TR069ToJSON 将TR069 XML数据转换为JSON格式
func (c *TR069Converter) TR069ToJSON(xmlData []byte) ([]byte, error) {
	// 使用解析器解析TR069消息
	message, err := c.parser.ParseMessage(context.Background(), xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TR069 message: %w", err)
	}

// 提取RPC方法和参数
	method := message.Method
	params := make(map[string]interface{})
	for _, param := range message.Parameters {
		params[param.Name] = param.Value
	}

	// 构建JSON响应
	response := map[string]interface{}{
		"method": method,
		"params": params,
	}

	// 添加额外信息
	if message.Fault != nil {
		response["fault"] = map[string]interface{}{
			"code":   message.Fault.FaultCode,
			"string": message.Fault.FaultString,
		}
}

// 转换为JSON
return json.Marshal(response)
}

// ParseTR069Fault 解析TR069错误信息
func (c *TR069Converter) ParseTR069Fault(xmlData []byte) (int, string, error) {
// 简单解析Fault结构
var envelope struct {
XMLName xml.Name `xml:"Envelope"`
Body    struct {
Fault struct {
FaultCode   int    `xml:"FaultCode"`
FaultString string `xml:"FaultString"`
} `xml:"Fault"`
} `xml:"Body"`
}

if err := xml.Unmarshal(xmlData, &envelope); err != nil {
return 0, "", err
}

return envelope.Body.Fault.FaultCode, envelope.Body.Fault.FaultString, nil
}

// BuildJSONFault 构建JSON错误响应
func (c *TR069Converter) BuildJSONFault(code int, message string) ([]byte, error) {
response := map[string]interface{}{
"success": false,
"error":   message,
"code":    code,
}

return json.Marshal(response)
}
