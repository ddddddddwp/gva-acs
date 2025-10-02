package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/types"
)

// ImprovedParser 改进的TR069解析器
type ImprovedParser struct {
	strictMode bool
}

// NewImproved 创建改进的解析器实例
func NewImproved() *ImprovedParser {
	return &ImprovedParser{
		strictMode: false,
	}
}

// SOAPEnvelope 改进的SOAP信封结构
type SOAPEnvelope struct {
	XMLName xml.Name   `xml:"Envelope"`
	Header  SOAPHeader `xml:"Header"`
	Body    SOAPBody   `xml:"Body"`
}

// SOAPHeader SOAP头部
type SOAPHeader struct {
	XMLName xml.Name `xml:"Header"`
	ID      string   `xml:"ID,attr,omitempty"`
}

// SOAPBody SOAP主体，支持多种RPC方法
type SOAPBody struct {
	XMLName xml.Name `xml:"Body"`
	// 使用interface{}来动态解析不同的RPC方法
	Content interface{} `xml:",any"`
}

// RawSOAPBody 用于获取原始XML内容的结构
type RawSOAPBody struct {
	XMLName xml.Name `xml:"Body"`
	Content []byte   `xml:",innerxml"`
}

// ParseMessage 解析TR069消息
func (p *ImprovedParser) ParseMessage(ctx context.Context, data []byte) (*interfaces.Message, error) {
	// 首先解析基本的SOAP结构
	var envelope SOAPEnvelope
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	// 获取RPC方法名
	method, err := p.ParseRPCMethod(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RPC method: %w", err)
	}

	// 创建消息对象
	message := &interfaces.Message{
		Method: method,
	}

	// 根据方法类型解析具体内容
	switch method {
	case "Inform":
		params, err := p.ParseParameterList(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
		message.Parameters = params
	default:
		// 对于其他方法，暂时不设置参数
		message.Parameters = []interfaces.Parameter{}
	}

	return message, nil
}

// ParseRPCMethod 改进的RPC方法解析
func (p *ImprovedParser) ParseRPCMethod(ctx context.Context, data []byte) (string, error) {
	// 获取SOAP Body的原始内容
	var envelope struct {
		XMLName xml.Name    `xml:"Envelope"`
		Body    RawSOAPBody `xml:"Body"`
	}
	
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("failed to unmarshal SOAP envelope: %w", err)
	}

	bodyContent := string(envelope.Body.Content)
	
	// 使用更精确的XML解析来确定方法名
	method, err := p.extractMethodFromXML(bodyContent)
	if err != nil {
		return "", fmt.Errorf("failed to extract method from XML: %w", err)
	}

	return method, nil
}

// extractMethodFromXML 从XML内容中提取方法名
func (p *ImprovedParser) extractMethodFromXML(xmlContent string) (string, error) {
	// 定义支持的TR069方法及其可能的命名空间前缀
	methods := map[string][]string{
		"Inform": {"Inform", "cwmp:Inform", "soap-env:Inform"},
		"GetRPCMethods": {"GetRPCMethods", "cwmp:GetRPCMethods"},
		"SetParameterValues": {"SetParameterValues", "cwmp:SetParameterValues"},
		"GetParameterValues": {"GetParameterValues", "cwmp:GetParameterValues"},
		"GetParameterNames": {"GetParameterNames", "cwmp:GetParameterNames"},
		"AddObject": {"AddObject", "cwmp:AddObject"},
		"DeleteObject": {"DeleteObject", "cwmp:DeleteObject"},
		"Download": {"Download", "cwmp:Download"},
		"Upload": {"Upload", "cwmp:Upload"},
		"Reboot": {"Reboot", "cwmp:Reboot"},
		"FactoryReset": {"FactoryReset", "cwmp:FactoryReset"},
		"InformResponse": {"InformResponse", "cwmp:InformResponse"},
	}

	// 查找匹配的方法标签
	for method, variants := range methods {
		for _, variant := range variants {
			// 查找开始标签
			startTag := fmt.Sprintf("<%s", variant)
			endTag := fmt.Sprintf("</%s>", variant)
			
			if strings.Contains(xmlContent, startTag) || strings.Contains(xmlContent, endTag) {
				return method, nil
			}
		}
	}

	// 如果没有找到已知方法，尝试从第一个标签提取
	if method := p.extractFirstTag(xmlContent); method != "" {
		return method, nil
	}

	return "", fmt.Errorf("unknown RPC method in XML content")
}

// extractFirstTag 从XML内容中提取第一个标签名
func (p *ImprovedParser) extractFirstTag(xmlContent string) string {
	// 简单的标签提取逻辑
	start := strings.Index(xmlContent, "<")
	if start == -1 {
		return ""
	}
	
	end := strings.Index(xmlContent[start:], ">")
	if end == -1 {
		return ""
	}
	
	tag := xmlContent[start+1 : start+end]
	
	// 移除命名空间前缀
	if colonIndex := strings.Index(tag, ":"); colonIndex != -1 {
		tag = tag[colonIndex+1:]
	}
	
	// 移除属性
	if spaceIndex := strings.Index(tag, " "); spaceIndex != -1 {
		tag = tag[:spaceIndex]
	}
	
	return tag
}

// parseInformMessage 解析Inform消息
func (p *ImprovedParser) parseInformMessage(data []byte) (*types.Inform, error) {
	// 首先尝试解析完整的SOAP信封
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			XMLName xml.Name     `xml:"Body"`
			Inform  types.Inform `xml:"Inform"`
		} `xml:"Body"`
	}
	
	if err := xml.Unmarshal(data, &envelope); err != nil {
		// 如果完整解析失败，尝试只解析Body内容
		var bodyEnvelope struct {
			XMLName xml.Name `xml:"Envelope"`
			Body    struct {
				XMLName xml.Name `xml:"Body"`
				Content []byte   `xml:",innerxml"`
			} `xml:"Body"`
		}
		
		if err := xml.Unmarshal(data, &bodyEnvelope); err != nil {
			return nil, fmt.Errorf("failed to parse SOAP envelope: %w", err)
		}
		
		// 尝试直接解析Inform内容
		var inform types.Inform
		if err := xml.Unmarshal(bodyEnvelope.Body.Content, &inform); err != nil {
			return nil, fmt.Errorf("failed to parse Inform content: %w", err)
		}
		
		return &inform, nil
	}
	
	return &envelope.Body.Inform, nil
}

// ParseParameterList 解析参数列表
func (p *ImprovedParser) ParseParameterList(ctx context.Context, data []byte) ([]interfaces.Parameter, error) {
	inform, err := p.parseInformMessage(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Inform message: %w", err)
	}
	
	var params []interfaces.Parameter
	for _, param := range inform.ParameterList.Parameters {
		params = append(params, param.ToInterface())
	}
	
	return params, nil
}

// SetStrictMode 设置严格模式
func (p *ImprovedParser) SetStrictMode(strict bool) {
	p.strictMode = strict
}

// GetStrictMode 获取严格模式状态
func (p *ImprovedParser) GetStrictMode() bool {
	return p.strictMode
}