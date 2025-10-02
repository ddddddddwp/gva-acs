package converter

import (
	"context"
	"encoding/json"
	"encoding/xml"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// JSONConverter 提供TR069消息和JSON格式之间的转换
type JSONConverter struct {
	parser  interfaces.Parser
	builder interfaces.Builder
}

// NewJSONConverter 创建新的JSON转换器
func NewJSONConverter(parser interfaces.Parser, builder interfaces.Builder) *JSONConverter {
	return &JSONConverter{
		parser:  parser,
		builder: builder,
	}
}

// TR069ToJSON 将TR069 XML消息转换为JSON格式
func (c *JSONConverter) TR069ToJSON(xmlData []byte) ([]byte, error) {
	// 使用解析器解析TR069消息
	message, err := c.parser.ParseMessage(context.Background(), xmlData)
	if err != nil {
		return nil, err
	}

	// 将消息转换为JSON
	return json.Marshal(message)
}

// JSONToTR069 将JSON格式转换为TR069 XML消息
func (c *JSONConverter) JSONToTR069(jsonData []byte) ([]byte, error) {
	// 解析JSON为消息结构
	var message interfaces.Message
	if err := json.Unmarshal(jsonData, &message); err != nil {
		return nil, err
	}

	// 使用构建器构建TR069消息
	return c.builder.BuildMessage(context.Background(), &message)
}

// ParseParametersToJSON 将TR069参数列表转换为JSON格式
func (c *JSONConverter) ParseParametersToJSON(xmlData []byte) ([]byte, error) {
	// 使用解析器解析参数列表
	params, err := c.parser.ParseParameterList(context.Background(), xmlData)
	if err != nil {
		return nil, err
	}

	// 将参数列表转换为JSON
	return json.Marshal(params)
}

// BuildParametersFromJSON 从JSON构建TR069参数列表
func (c *JSONConverter) BuildParametersFromJSON(jsonData []byte) ([]byte, error) {
	// 解析JSON为参数列表
	var params []interfaces.Parameter
	if err := json.Unmarshal(jsonData, &params); err != nil {
		return nil, err
	}

	// 构建参数列表XML
	// 这里简化处理，实际应用中可能需要更复杂的逻辑
	type ParameterList struct {
		XMLName    xml.Name               `xml:"ParameterList"`
		Parameters []interfaces.Parameter `xml:"Parameter"`
	}

	paramList := ParameterList{
		Parameters: params,
	}

	return xml.Marshal(paramList)
}

// BuildFaultFromJSON 从JSON构建TR069故障消息
func (c *JSONConverter) BuildFaultFromJSON(jsonData []byte) ([]byte, error) {
	// 解析JSON为故障信息
	var fault struct {
		FaultCode   int    `json:"faultCode"`
		FaultString string `json:"faultString"`
	}
	if err := json.Unmarshal(jsonData, &fault); err != nil {
		return nil, err
	}

	// 使用构建器构建故障消息
	return c.builder.BuildFault(context.Background(), fault.FaultCode, fault.FaultString)
}