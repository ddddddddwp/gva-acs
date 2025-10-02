package core

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/interfaces"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SOAPHandler SOAP消息处理器
type SOAPHandler struct {
	logger         *zap.Logger
	sessionManager interfaces.TR069SessionManager
	eventHandler   interfaces.TR069EventHandler
	parser         interfaces.TR069Parser
	builder        interfaces.TR069Builder
}

// NewSOAPHandler 创建SOAP处理器
func NewSOAPHandler(
	logger *zap.Logger,
	sessionManager interfaces.TR069SessionManager,
	eventHandler interfaces.TR069EventHandler,
	parser interfaces.TR069Parser,
	builder interfaces.TR069Builder,
) *SOAPHandler {
	return &SOAPHandler{
		logger:         logger,
		sessionManager: sessionManager,
		eventHandler:   eventHandler,
		parser:         parser,
		builder:        builder,
	}
}

// HandleSOAPRequest 处理SOAP请求
func (h *SOAPHandler) HandleSOAPRequest(c *gin.Context) {
	ctx := c.Request.Context()
	
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.XML(http.StatusBadRequest, h.buildSOAPFault("Client", "Invalid request body"))
		return
	}
	
	// 记录接收到的SOAP消息
	h.logger.Debug("Received SOAP message", 
		zap.String("client_ip", c.ClientIP()),
		zap.String("body", string(body)))
	
	// 解析SOAP消息
	message, err := h.parser.ParseSOAPMessage(ctx, body)
	if err != nil {
		h.logger.Error("Failed to parse SOAP message", zap.Error(err))
		c.XML(http.StatusBadRequest, h.buildSOAPFault("Client", "Invalid SOAP message"))
		return
	}
	
	// 获取或创建会话
	session, err := h.getOrCreateSession(ctx, c.ClientIP(), message)
	if err != nil {
		h.logger.Error("Failed to get or create session", zap.Error(err))
		c.XML(http.StatusInternalServerError, h.buildSOAPFault("Server", "Session management error"))
		return
	}
	
	// 处理消息
	response, err := h.processMessage(ctx, session, message)
	if err != nil {
		h.logger.Error("Failed to process message", zap.Error(err))
		c.XML(http.StatusInternalServerError, h.buildSOAPFault("Server", "Message processing error"))
		return
	}
	
	// 构建SOAP响应
	soapResponse := h.buildSOAPResponse(response)
	
	// 记录发送的SOAP响应
	h.logger.Debug("Sending SOAP response", 
		zap.String("session_id", session.SessionID),
		zap.Any("response", soapResponse))
	
	// 设置响应头
	c.Header("Content-Type", "text/xml; charset=utf-8")
	c.Header("SOAPAction", "")
	
	// 发送响应
	c.XML(http.StatusOK, soapResponse)
}

// getOrCreateSession 获取或创建会话
func (h *SOAPHandler) getOrCreateSession(ctx context.Context, clientIP string, message *interfaces.Message) (*interfaces.Session, error) {
	// 尝试从消息中提取设备ID
	deviceID := h.extractDeviceID(message)
	if deviceID == "" {
		return nil, fmt.Errorf("cannot extract device ID from message")
	}
	
	// 查找现有会话
	sessions, err := h.sessionManager.GetSession(ctx, deviceID)
	if err == nil && sessions != nil && sessions.Status == interfaces.SessionStatusActive {
		// 更新会话活动时间
		sessions.LastActivity = time.Now()
		err = h.sessionManager.UpdateSession(ctx, sessions)
		if err != nil {
			h.logger.Warn("Failed to update session activity", zap.Error(err))
		}
		return sessions, nil
	}
	
	// 创建新会话
	return h.sessionManager.CreateSession(ctx, deviceID, clientIP)
}

// extractDeviceID 从消息中提取设备ID
func (h *SOAPHandler) extractDeviceID(message *interfaces.Message) string {
	if message == nil || message.Params == nil {
		return ""
	}
	
	// 尝试从不同的参数中提取设备ID
	if deviceInfo, ok := message.Params["DeviceId"]; ok {
		if deviceMap, ok := deviceInfo.(map[string]interface{}); ok {
			if serialNumber, ok := deviceMap["SerialNumber"].(string); ok {
				return serialNumber
			}
		}
	}
	
	// 尝试从参数列表中提取
	if paramList, ok := message.Params["ParameterList"]; ok {
		if params, ok := paramList.([]interface{}); ok {
			for _, param := range params {
				if paramMap, ok := param.(map[string]interface{}); ok {
					if name, ok := paramMap["Name"].(string); ok {
						if strings.Contains(name, "SerialNumber") {
							if value, ok := paramMap["Value"].(string); ok {
								return value
							}
						}
					}
				}
			}
		}
	}
	
	return ""
}

// processMessage 处理消息
func (h *SOAPHandler) processMessage(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	switch message.Method {
	case "Inform":
		return h.processInform(ctx, session, message)
	case "GetParameterValuesResponse":
		return h.processGetParameterValuesResponse(ctx, session, message)
	case "SetParameterValuesResponse":
		return h.processSetParameterValuesResponse(ctx, session, message)
	case "TransferComplete":
		return h.processTransferComplete(ctx, session, message)
	case "GetRPCMethods":
		return h.processGetRPCMethods(ctx, session, message)
	default:
		return nil, fmt.Errorf("unsupported method: %s", message.Method)
	}
}

// processInform 处理Inform消息
func (h *SOAPHandler) processInform(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	// 解析Inform消息
	informData, err := xml.Marshal(message.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inform params: %w", err)
	}
	
	inform, err := h.parser.ParseInform(ctx, informData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inform message: %w", err)
	}
	
	// 处理Inform事件
	response, err := h.eventHandler.HandleInform(ctx, session, inform)
	if err != nil {
		return nil, fmt.Errorf("failed to handle inform: %w", err)
	}
	
	return response, nil
}

// processGetParameterValuesResponse 处理获取参数值响应
func (h *SOAPHandler) processGetParameterValuesResponse(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	// 解析响应数据
	responseData, err := xml.Marshal(message.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response params: %w", err)
	}
	
	response, err := h.parser.ParseGetParameterValuesResponse(ctx, responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse get parameter values response: %w", err)
	}
	
	// 处理响应
	return h.eventHandler.HandleGetParameterValues(ctx, session, &interfaces.GetParameterValuesRequest{
		ParameterNames: h.extractParameterNames(response.ParameterList),
	})
}

// processSetParameterValuesResponse 处理设置参数值响应
func (h *SOAPHandler) processSetParameterValuesResponse(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	// 解析响应数据
	responseData, err := xml.Marshal(message.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response params: %w", err)
	}
	
	response, err := h.parser.ParseSetParameterValuesResponse(ctx, responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse set parameter values response: %w", err)
	}
	
	// 处理响应
	return h.eventHandler.HandleSetParameterValues(ctx, session, &interfaces.SetParameterValuesRequest{
		ParameterList: h.extractParameters(message.Params),
	})
}

// processTransferComplete 处理传输完成消息
func (h *SOAPHandler) processTransferComplete(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	// 解析传输完成数据
	responseData, err := xml.Marshal(message.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer complete params: %w", err)
	}
	
	response, err := h.parser.ParseTransferCompleteResponse(ctx, responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transfer complete response: %w", err)
	}
	
	// 构建传输完成请求
	request := &interfaces.TransferCompleteRequest{
		CommandKey:   h.extractCommandKey(message.Params),
		FaultCode:    0,
		FaultString:  "",
		StartTime:    time.Now(),
		CompleteTime: time.Now(),
	}
	
	// 处理传输完成事件
	return h.eventHandler.HandleTransferComplete(ctx, session, request)
}

// processGetRPCMethods 处理获取RPC方法消息
func (h *SOAPHandler) processGetRPCMethods(ctx context.Context, session *interfaces.Session, message *interfaces.Message) (interface{}, error) {
	// 返回支持的RPC方法列表
	methods := []string{
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterNames",
		"SetParameterAttributes",
		"GetParameterAttributes",
		"AddObject",
		"DeleteObject",
		"Reboot",
		"FactoryReset",
		"Download",
		"Upload",
	}
	
	return map[string]interface{}{
		"MethodList": methods,
	}, nil
}

// extractParameterNames 从参数列表中提取参数名称
func (h *SOAPHandler) extractParameterNames(parameters []interfaces.Parameter) []string {
	names := make([]string, len(parameters))
	for i, param := range parameters {
		names[i] = param.Name
	}
	return names
}

// extractParameters 从消息参数中提取参数列表
func (h *SOAPHandler) extractParameters(params map[string]interface{}) []interfaces.Parameter {
	var parameters []interfaces.Parameter
	
	if paramList, ok := params["ParameterList"]; ok {
		if paramArray, ok := paramList.([]interface{}); ok {
			for _, param := range paramArray {
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
	
	return parameters
}

// extractCommandKey 从消息参数中提取命令键
func (h *SOAPHandler) extractCommandKey(params map[string]interface{}) string {
	if commandKey, ok := params["CommandKey"]; ok {
		if key, ok := commandKey.(string); ok {
			return key
		}
	}
	return ""
}

// buildSOAPResponse 构建SOAP响应
func (h *SOAPHandler) buildSOAPResponse(response interface{}) *SOAPEnvelope {
	return &SOAPEnvelope{
		XMLName: xml.Name{Space: "http://schemas.xmlsoap.org/soap/envelope/", Local: "Envelope"},
		Header:  &SOAPHeader{},
		Body: &SOAPBody{
			Content: response,
		},
	}
}

// buildSOAPFault 构建SOAP错误响应
func (h *SOAPHandler) buildSOAPFault(faultCode, faultString string) *SOAPEnvelope {
	return &SOAPEnvelope{
		XMLName: xml.Name{Space: "http://schemas.xmlsoap.org/soap/envelope/", Local: "Envelope"},
		Header:  &SOAPHeader{},
		Body: &SOAPBody{
			Fault: &SOAPFault{
				FaultCode:   faultCode,
				FaultString: faultString,
			},
		},
	}
}

// SOAP结构定义

// SOAPEnvelope SOAP信封
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"soap:Envelope"`
	Header  *SOAPHeader `xml:"soap:Header,omitempty"`
	Body    *SOAPBody   `xml:"soap:Body"`
}

// SOAPHeader SOAP头部
type SOAPHeader struct {
	ID           string `xml:"cwmp:ID,omitempty"`
	HoldRequests bool   `xml:"cwmp:HoldRequests,omitempty"`
}

// SOAPBody SOAP主体
type SOAPBody struct {
	Content interface{} `xml:",omitempty"`
	Fault   *SOAPFault  `xml:"soap:Fault,omitempty"`
}

// SOAPFault SOAP错误
type SOAPFault struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
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

// SerializeSOAPMessage 序列化SOAP消息
func SerializeSOAPMessage(envelope *SOAPEnvelope) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	
	err := encoder.Encode(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to encode SOAP envelope: %w", err)
	}
	
	return buf.Bytes(), nil
}