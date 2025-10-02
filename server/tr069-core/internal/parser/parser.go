// Package parser implements the TR069 message parser.
// 包 parser 实现了 TR069 消息解析器。
package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/rpc"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/types"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/version"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/crypto/encryption"
)

// Parser implements the interfaces.Parser interface.
// Parser 实现了 interfaces.Parser 接口。
type Parser struct {
	strictMode     bool
	securityConfig *types.SecurityConfig
	rpcRegistry    *rpc.Registry
}

// New creates a new Parser instance.
// New 创建一个新的 Parser 实例。
func New() interfaces.Parser {
	return &Parser{
		strictMode:  false,
		rpcRegistry: rpc.NewRegistry(),
	}
}

// ParseMessage parses a TR069 message from raw bytes.
// ParseMessage 从原始字节数据中解析 TR069 消息。
func (p *Parser) ParseMessage(ctx context.Context, data []byte) (*interfaces.Message, error) {
	// Verify signature if security is enabled
	if p.securityConfig != nil && p.securityConfig.EnableSignatureVerification {
		if err := p.verifyMessageSignature(data, p.securityConfig.SignatureKey); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}
	
	// Parse the XML message
	var envelope types.Envelope
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}
	
	// Create the message
	msg := &interfaces.Message{
		Method: p.getMethodName(envelope),
		ID:     envelope.Header.ID,
	}
	
	// Parse specific message types
	switch msg.Method {
	case "Inform":
		inform, err := p.parseInformMessage(envelope.Body.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Inform message: %w", err)
		}
		msg.DeviceID = &interfaces.DeviceID{
			Manufacturer: inform.DeviceId.Manufacturer,
			OUI:          inform.DeviceId.OUI,
			ProductClass: inform.DeviceId.ProductClass,
			SerialNumber: inform.DeviceId.SerialNumber,
		}
		msg.Events = make([]interfaces.Event, len(inform.Event.Events))
		for i, event := range inform.Event.Events {
			msg.Events[i] = interfaces.Event{
				EventCode:  event.EventCode,
				CommandKey: event.CommandKey,
			}
		}
		msg.MaxEnvelopes = inform.MaxEnvelopes
		msg.CurrentTime = inform.CurrentTime
		msg.RetryCount = inform.RetryCount
		
		// Convert parameters
		msg.Parameters = make([]interfaces.Parameter, len(inform.ParameterList.Parameters))
		for i, param := range inform.ParameterList.Parameters {
			msg.Parameters[i] = interfaces.Parameter{
				Name:  param.Name,
				Value: param.Value.Value,
				Type:  param.Value.Type,
			}
		}
	default:
		// Parse parameters for other message types
		params, err := p.ParseParameterList(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter list: %w", err)
		}
		msg.Parameters = params
	}
	
	return msg, nil
}

// ParseRPCMethod extracts the RPC method from the raw message.
// ParseRPCMethod 从原始消息中提取 RPC 方法。
func (p *Parser) ParseRPCMethod(ctx context.Context, data []byte) (string, error) {
	var envelope types.Envelope
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("failed to unmarshal XML: %w", err)
	}
	
	return p.getMethodName(envelope), nil
}

// ParseParameterList extracts the parameter list from the raw message.
// ParseParameterList 从原始消息中提取参数列表。
func (p *Parser) ParseParameterList(ctx context.Context, data []byte) ([]interfaces.Parameter, error) {
	var envelope types.Envelope
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}
	
	var params []interfaces.Parameter
	
	// Parse parameters from the body content
	bodyContent := string(envelope.Body.Contents)
	
	// Check if this is an Inform message with ParameterList
	if strings.Contains(bodyContent, "<ParameterList>") {
		// Extract individual ParameterValueStruct elements
		for {
			start := strings.Index(bodyContent, "<ParameterValueStruct>")
			if start < 0 {
				break
			}
			
			end := strings.Index(bodyContent[start:], "</ParameterValueStruct>") + len("</ParameterValueStruct>")
			if end <= len("</ParameterValueStruct>") {
				break
			}
			
			paramXML := bodyContent[start : start+end]
			bodyContent = bodyContent[start+end:]
			
			// Parse individual parameter
			var param struct {
				Name  string `xml:"Name"`
				Value struct {
					Type  string `xml:"type,attr"`
					Value string `xml:",chardata"`
				} `xml:"Value"`
			}
			
			if err := xml.Unmarshal([]byte("<Root>"+paramXML+"</Root>"), &param); err == nil {
				// Decrypt sensitive parameter if needed
				value := param.Value.Value
				if p.securityConfig != nil && p.securityConfig.EnableParameterEncryption {
					if p.securityConfig.IsSensitiveParameter(param.Name) {
						decrypted, err := p.decryptSensitiveParameter(value, p.securityConfig.EncryptionKey)
						if err != nil {
							return nil, fmt.Errorf("failed to decrypt parameter %s: %w", param.Name, err)
						}
						// Type assert the decrypted value to string
						if str, ok := decrypted.(string); ok {
							value = str
						}
					}
				}
				
				params = append(params, interfaces.Parameter{
					Name:  param.Name,
					Value: value,
					Type:  param.Value.Type,
				})
			}
		}
	}
	
	return params, nil
}

// SetStrictMode sets the parser to strict mode, which will enforce stricter validation.
// SetStrictMode 设置解析器的严格模式，启用更严格的验证。
func (p *Parser) SetStrictMode(strict bool) {
	p.strictMode = strict
}

// GetStrictMode returns the current strict mode setting.
// GetStrictMode 返回当前的严格模式设置。
func (p *Parser) GetStrictMode() bool {
	return p.strictMode
}

// SetSecurityConfig sets the security configuration for the parser.
// SetSecurityConfig 为解析器设置安全配置。
func (p *Parser) SetSecurityConfig(config *types.SecurityConfig) {
	p.securityConfig = config
}

// getMethodName extracts the method name from the envelope.
// getMethodName 从信封中提取方法名称。
func (p *Parser) getMethodName(envelope types.Envelope) string {
	// Extract the actual method name from the body content
	bodyContent := string(envelope.Body.Contents)
	
	// Look for known TR-069 methods in the body
	methods := []string{"Inform", "GetRPCMethods", "SetParameterValues", "GetParameterValues", 
		"AddObject", "DeleteObject", "Download", "Upload", "Reboot", "FactoryReset"}
	
	for _, method := range methods {
		if strings.Contains(bodyContent, method) {
			return method
		}
	}
	
	// Default to Inform if no method found
	return "Inform"
}

// verifyMessageSignature verifies the message signature.
// verifyMessageSignature 验证消息签名。
func (p *Parser) verifyMessageSignature(data []byte, key []byte) error {
	// Implementation would verify the signature of the message
	// 这里会实现消息签名的验证逻辑
	return nil
}

// decryptSensitiveParameter decrypts a sensitive parameter.
// decryptSensitiveParameter 解密敏感参数。
func (p *Parser) decryptSensitiveParameter(value interface{}, key []byte) (interface{}, error) {
	// Implementation would decrypt the parameter value
	// 这里会实现参数值的解密逻辑
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	
	decrypted, err := encryption.DecryptParameter([]byte(str), key)
	if err != nil {
		return value, err
	}
	
	return string(decrypted), nil
}

// DetectVersion detects the TR-069 protocol version from the message.
// DetectVersion 从消息中检测 TR-069 协议版本。
func (p *Parser) DetectVersion(data []byte) interfaces.Version {
	v := version.DetectVersion(data)
	return interfaces.Version{
		Major: v.Major,
		Minor: v.Minor,
	}
}

// IsVersionSupported checks if the given version is supported.
// IsVersionSupported 检查给定版本是否受支持。
func (p *Parser) IsVersionSupported(v interfaces.Version) bool {
	ver := version.Version{Major: v.Major, Minor: v.Minor}
	return version.IsVersionSupported(ver)
}

// RegisterCustomMethod registers a custom RPC method handler.
// RegisterCustomMethod 注册自定义 RPC 方法处理器。
func (p *Parser) RegisterCustomMethod(method string, handler func(ctx context.Context, params map[string]interface{}) (interface{}, error)) error {
	return p.rpcRegistry.Register(method, handler)
}

// UnregisterCustomMethod unregisters a custom RPC method handler.
// UnregisterCustomMethod 取消注册自定义 RPC 方法处理器。
func (p *Parser) UnregisterCustomMethod(method string) error {
	return p.rpcRegistry.Unregister(method)
}

// ListCustomMethods returns a list of registered custom methods.
// ListCustomMethods 返回已注册的自定义方法列表。
func (p *Parser) ListCustomMethods() []string {
	return p.rpcRegistry.ListMethods()
}

// IsCustomMethodRegistered checks if a custom method is registered.
// IsCustomMethodRegistered 检查自定义方法是否已注册。
func (p *Parser) IsCustomMethodRegistered(method string) bool {
	return p.rpcRegistry.IsRegistered(method)
}

// parseInformMessage parses an Inform message from XML content.
// parseInformMessage 从 XML 内容中解析 Inform 消息。
func (p *Parser) parseInformMessage(xmlContent []byte) (*types.Inform, error) {
	var inform types.Inform
	if err := xml.Unmarshal(xmlContent, &inform); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Inform message: %w", err)
	}
	return &inform, nil
}

// parseParameterList parses parameter list from XML content with enhanced type support
func (p *Parser) parseParameterList(xmlContent []byte) ([]interfaces.Parameter, error) {
	var envelope types.Envelope
	if err := xml.Unmarshal(xmlContent, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	var paramList types.ParameterValueList
	if err := xml.Unmarshal([]byte(envelope.Body.Contents), &paramList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameter list: %w", err)
	}

	parameters := make([]interfaces.Parameter, len(paramList.Parameters))
	for i, param := range paramList.Parameters {
		// Extract type information from Value's type attribute
		paramType := param.Value.Type
		if paramType == "" {
			paramType = "xsd:string" // default type
		}
		
		parameters[i] = interfaces.Parameter{
			Name:  param.Name,
			Value: param.Value.Value,
			Type:  paramType,
		}
	}

	return parameters, nil
}
