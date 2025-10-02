package core

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/interfaces"
	"go.uber.org/zap"
)

// TR069Parser TR069消息解析器实现
type TR069Parser struct {
	logger *zap.Logger
}

// NewTR069Parser 创建TR069解析器
func NewTR069Parser(logger *zap.Logger) *TR069Parser {
	return &TR069Parser{
		logger: logger,
	}
}

// ParseSOAPMessage 解析SOAP消息
func (p *TR069Parser) ParseSOAPMessage(ctx context.Context, data []byte) (*interfaces.Message, error) {
	var envelope SOAPEnvelope
	err := xml.Unmarshal(data, &envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal SOAP envelope: %w", err)
	}
	
	// 提取方法名和参数
	method, params, err := p.extractMethodAndParams(&envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to extract method and params: %w", err)
	}
	
	// 提取消息ID
	id := ""
	if envelope.Header != nil {
		id = envelope.Header.ID
	}
	
	return &interfaces.Message{
		Method: method,
		ID:     id,
		Params: params,
	}, nil
}

// ParseInform 解析Inform消息
func (p *TR069Parser) ParseInform(ctx context.Context, data []byte) (*interfaces.InformMessage, error) {
	var informStruct struct {
		DeviceID      DeviceIDStruct    `xml:"DeviceId"`
		Event         []EventStruct     `xml:"Event>EventStruct"`
		MaxEnvelopes  int               `xml:"MaxEnvelopes"`
		CurrentTime   string            `xml:"CurrentTime"`
		RetryCount    int               `xml:"RetryCount"`
		ParameterList []ParameterStruct `xml:"ParameterList>ParameterValueStruct"`
	}
	
	err := xml.Unmarshal(data, &informStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal inform message: %w", err)
	}
	
	// 转换设备ID
	deviceID := interfaces.DeviceID{
		Manufacturer: informStruct.DeviceID.Manufacturer,
		OUI:          informStruct.DeviceID.OUI,
		ProductClass: informStruct.DeviceID.ProductClass,
		SerialNumber: informStruct.DeviceID.SerialNumber,
	}
	
	// 转换事件列表
	events := make([]interfaces.EventStruct, len(informStruct.Event))
	for i, event := range informStruct.Event {
		events[i] = interfaces.EventStruct{
			EventCode:  event.EventCode,
			CommandKey: event.CommandKey,
		}
	}
	
	// 转换参数列表
	parameters := make([]interfaces.Parameter, len(informStruct.ParameterList))
	for i, param := range informStruct.ParameterList {
		parameters[i] = interfaces.Parameter{
			Name:      param.Name,
			Value:     param.Value,
			Type:      param.Type,
			Writable:  true, // 默认可写
			Timestamp: time.Now(),
		}
	}
	
	return &interfaces.InformMessage{
		DeviceID:      deviceID,
		Event:         events,
		MaxEnvelopes:  informStruct.MaxEnvelopes,
		CurrentTime:   informStruct.CurrentTime,
		RetryCount:    informStruct.RetryCount,
		ParameterList: parameters,
	}, nil
}

// ParseGetParameterValuesResponse 解析获取参数值响应
func (p *TR069Parser) ParseGetParameterValuesResponse(ctx context.Context, data []byte) (*interfaces.GetParameterValuesResponse, error) {
	var responseStruct struct {
		ParameterList []ParameterStruct `xml:"ParameterList>ParameterValueStruct"`
	}
	
	err := xml.Unmarshal(data, &responseStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal get parameter values response: %w", err)
	}
	
	// 转换参数列表
	parameters := make([]interfaces.Parameter, len(responseStruct.ParameterList))
	for i, param := range responseStruct.ParameterList {
		parameters[i] = interfaces.Parameter{
			Name:      param.Name,
			Value:     param.Value,
			Type:      param.Type,
			Writable:  true,
			Timestamp: time.Now(),
		}
	}
	
	return &interfaces.GetParameterValuesResponse{
		ParameterList: parameters,
	}, nil
}

// ParseSetParameterValuesResponse 解析设置参数值响应
func (p *TR069Parser) ParseSetParameterValuesResponse(ctx context.Context, data []byte) (*interfaces.SetParameterValuesResponse, error) {
	var responseStruct struct {
		Status int `xml:"Status"`
	}
	
	err := xml.Unmarshal(data, &responseStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal set parameter values response: %w", err)
	}
	
	return &interfaces.SetParameterValuesResponse{
		Status: responseStruct.Status,
	}, nil
}

// ParseTransferCompleteResponse 解析传输完成响应
func (p *TR069Parser) ParseTransferCompleteResponse(ctx context.Context, data []byte) (*interfaces.TransferCompleteResponse, error) {
	var responseStruct struct {
		Status int `xml:"Status"`
	}
	
	err := xml.Unmarshal(data, &responseStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transfer complete response: %w", err)
	}
	
	return &interfaces.TransferCompleteResponse{
		Status: responseStruct.Status,
	}, nil
}

// extractMethodAndParams 从SOAP信封中提取方法名和参数
func (p *TR069Parser) extractMethodAndParams(envelope *SOAPEnvelope) (string, map[string]interface{}, error) {
	if envelope.Body == nil {
		return "", nil, fmt.Errorf("SOAP body is empty")
	}
	
	// 将Body内容转换为字节数组进行进一步解析
	bodyData, err := xml.Marshal(envelope.Body.Content)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal body content: %w", err)
	}
	
	// 尝试解析不同类型的消息
	method, params := p.parseMessageContent(bodyData)
	
	return method, params, nil
}

// parseMessageContent 解析消息内容
func (p *TR069Parser) parseMessageContent(data []byte) (string, map[string]interface{}) {
	content := string(data)
	
	// 检测消息类型
	if strings.Contains(content, "Inform") {
		return "Inform", p.parseInformContent(data)
	} else if strings.Contains(content, "GetParameterValuesResponse") {
		return "GetParameterValuesResponse", p.parseGetParameterValuesResponseContent(data)
	} else if strings.Contains(content, "SetParameterValuesResponse") {
		return "SetParameterValuesResponse", p.parseSetParameterValuesResponseContent(data)
	} else if strings.Contains(content, "TransferComplete") {
		return "TransferComplete", p.parseTransferCompleteContent(data)
	} else if strings.Contains(content, "GetRPCMethods") {
		return "GetRPCMethods", make(map[string]interface{})
	}
	
	// 默认返回空参数
	return "Unknown", make(map[string]interface{})
}

// parseInformContent 解析Inform消息内容
func (p *TR069Parser) parseInformContent(data []byte) map[string]interface{} {
	var informStruct struct {
		DeviceID      DeviceIDStruct    `xml:"DeviceId"`
		Event         []EventStruct     `xml:"Event>EventStruct"`
		MaxEnvelopes  int               `xml:"MaxEnvelopes"`
		CurrentTime   string            `xml:"CurrentTime"`
		RetryCount    int               `xml:"RetryCount"`
		ParameterList []ParameterStruct `xml:"ParameterList>ParameterValueStruct"`
	}
	
	err := xml.Unmarshal(data, &informStruct)
	if err != nil {
		p.logger.Error("Failed to parse inform content", zap.Error(err))
		return make(map[string]interface{})
	}
	
	params := make(map[string]interface{})
	params["DeviceId"] = map[string]interface{}{
		"Manufacturer": informStruct.DeviceID.Manufacturer,
		"OUI":          informStruct.DeviceID.OUI,
		"ProductClass": informStruct.DeviceID.ProductClass,
		"SerialNumber": informStruct.DeviceID.SerialNumber,
	}
	params["Event"] = informStruct.Event
	params["MaxEnvelopes"] = informStruct.MaxEnvelopes
	params["CurrentTime"] = informStruct.CurrentTime
	params["RetryCount"] = informStruct.RetryCount
	params["ParameterList"] = informStruct.ParameterList
	
	return params
}

// parseGetParameterValuesResponseContent 解析获取参数值响应内容
func (p *TR069Parser) parseGetParameterValuesResponseContent(data []byte) map[string]interface{} {
	var responseStruct struct {
		ParameterList []ParameterStruct `xml:"ParameterList>ParameterValueStruct"`
	}
	
	err := xml.Unmarshal(data, &responseStruct)
	if err != nil {
		p.logger.Error("Failed to parse get parameter values response content", zap.Error(err))
		return make(map[string]interface{})
	}
	
	params := make(map[string]interface{})
	params["ParameterList"] = responseStruct.ParameterList
	
	return params
}

// parseSetParameterValuesResponseContent 解析设置参数值响应内容
func (p *TR069Parser) parseSetParameterValuesResponseContent(data []byte) map[string]interface{} {
	var responseStruct struct {
		Status int `xml:"Status"`
	}
	
	err := xml.Unmarshal(data, &responseStruct)
	if err != nil {
		p.logger.Error("Failed to parse set parameter values response content", zap.Error(err))
		return make(map[string]interface{})
	}
	
	params := make(map[string]interface{})
	params["Status"] = responseStruct.Status
	
	return params
}

// parseTransferCompleteContent 解析传输完成内容
func (p *TR069Parser) parseTransferCompleteContent(data []byte) map[string]interface{} {
	var transferStruct struct {
		CommandKey   string `xml:"CommandKey"`
		FaultCode    int    `xml:"FaultStruct>FaultCode"`
		FaultString  string `xml:"FaultStruct>FaultString"`
		StartTime    string `xml:"StartTime"`
		CompleteTime string `xml:"CompleteTime"`
	}
	
	err := xml.Unmarshal(data, &transferStruct)
	if err != nil {
		p.logger.Error("Failed to parse transfer complete content", zap.Error(err))
		return make(map[string]interface{})
	}
	
	params := make(map[string]interface{})
	params["CommandKey"] = transferStruct.CommandKey
	params["FaultCode"] = transferStruct.FaultCode
	params["FaultString"] = transferStruct.FaultString
	params["StartTime"] = transferStruct.StartTime
	params["CompleteTime"] = transferStruct.CompleteTime
	
	return params
}

// XML结构定义

// DeviceIDStruct 设备ID结构
type DeviceIDStruct struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

// EventStruct 事件结构
type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

// ParameterStruct 参数结构
type ParameterStruct struct {
	Name  string      `xml:"Name"`
	Value interface{} `xml:"Value"`
	Type  string      `xml:"Type,attr"`
}

// FaultStruct 错误结构
type FaultStruct struct {
	FaultCode   int    `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// 辅助函数

// parseValue 解析参数值
func (p *TR069Parser) parseValue(value interface{}, valueType string) interface{} {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return value
	}
	
	switch strings.ToLower(valueType) {
	case "xsd:int", "int":
		if intVal, err := strconv.Atoi(str); err == nil {
			return intVal
		}
	case "xsd:boolean", "boolean":
		if boolVal, err := strconv.ParseBool(str); err == nil {
			return boolVal
		}
	case "xsd:datetime", "datetime":
		if timeVal, err := time.Parse(time.RFC3339, str); err == nil {
			return timeVal
		}
	case "xsd:long", "long":
		if longVal, err := strconv.ParseInt(str, 10, 64); err == nil {
			return longVal
		}
	case "xsd:unsignedint", "unsignedint":
		if uintVal, err := strconv.ParseUint(str, 10, 32); err == nil {
			return uint32(uintVal)
		}
	}
	
	return str
}

// validateMessage 验证消息格式
func (p *TR069Parser) validateMessage(message *interfaces.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	
	if message.Method == "" {
		return fmt.Errorf("message method is empty")
	}
	
	return nil
}