// Package builder implements the TR069 message builder.
// 包 builder 实现了 TR069 消息构建器。
package builder

import (
	"context"
	"encoding/xml"
	"fmt"
	
	interfaces "github.com/root/demo/tr069/interfaces"
	"github.com/root/demo/tr069/internal/types"
)

// Builder implements the interfaces.Builder interface.
// Builder 实现了 interfaces.Builder 接口。
type Builder struct {
	prettyPrint    bool
	securityConfig *types.SecurityConfig
}

// New creates a new Builder instance.
// New 创建一个新的 Builder 实例。
func New() interfaces.Builder {
	return &Builder{
		prettyPrint: false,
	}
}

// BuildMessage builds a TR069 message from a Message struct.
// BuildMessage 从 Message 结构体构建 TR069 消息。
func (b *Builder) BuildMessage(ctx context.Context, msg *interfaces.Message) ([]byte, error) {
	// Create envelope with proper namespaces
	envelope := struct {
		XMLName xml.Name `xml:"SOAP-ENV:Envelope"`
		SoapEnv string   `xml:"xmlns:SOAP-ENV,attr"`
		SoapEnc string   `xml:"xmlns:SOAP-ENC,attr"`
		XSD     string   `xml:"xmlns:xsd,attr"`
		XSI     string   `xml:"xmlns:xsi,attr"`
		CWMP    string   `xml:"xmlns:cwmp,attr"`
		Header  struct {
			ID             string `xml:"ID"`
			SessionID      string `xml:"SessionID,omitempty"`
			HoldRequests   bool   `xml:"HoldRequests,omitempty"`
			NoMoreRequests int    `xml:"NoMoreRequests,omitempty"`
		} `xml:"SOAP-ENV:Header"`
		Body    struct {
			XMLName  xml.Name `xml:"SOAP-ENV:Body"`
			Contents []byte   `xml:",innerxml"`
		} `xml:"SOAP-ENV:Body"`
	}{
		SoapEnv: "http://schemas.xmlsoap.org/soap/envelope/",
		SoapEnc: "http://schemas.xmlsoap.org/soap/encoding/",
		XSD:     "http://www.w3.org/2001/XMLSchema",
		XSI:     "http://www.w3.org/2001/XMLSchema-instance",
		CWMP:    "urn:dslforum-org:cwmp-1-0",
		Header: struct {
			ID             string `xml:"ID"`
			SessionID      string `xml:"SessionID,omitempty"`
			HoldRequests   bool   `xml:"HoldRequests,omitempty"`
			NoMoreRequests int    `xml:"NoMoreRequests,omitempty"`
		}{
			ID:             msg.ID,
			NoMoreRequests: 0,
		},
	}
	
	// Handle fault messages
	if msg.Fault != nil {
		fault := types.Fault{
			FaultCode:   msg.Fault.FaultCode,
			FaultString: msg.Fault.FaultString,
		}
		faultXML, err := xml.Marshal(fault)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal fault: %w", err)
		}
		envelope.Body.Contents = faultXML
	} else {
		// Handle regular messages based on method
		switch msg.Method {
		case "Inform":
			// For Inform method, we typically respond with InformResponse
			informResponse := types.InformResponse{
				MaxEnvelopes: 1,
			}
			responseXML, err := xml.Marshal(informResponse)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal inform response: %w", err)
			}
			envelope.Body.Contents = responseXML
		default:
			// For other methods, create parameter list if parameters exist
			if len(msg.Parameters) > 0 {
				parameterList := types.ParameterValueList{
					Parameters: make([]types.ParameterValueStruct, len(msg.Parameters)),
				}
				
				for i, param := range msg.Parameters {
					parameterList.Parameters[i] = types.ParameterValueStruct{
						Name: param.Name,
						Value: types.Value{
							Type:  param.Type,
							Value: fmt.Sprintf("%v", param.Value),
						},
					}
				}
				
				// Create a simple response with the parameters
				response := struct {
					XMLName       xml.Name              `xml:"Response"`
					ParameterList types.ParameterValueList `xml:"ParameterList"`
				}{
					ParameterList: parameterList,
				}
				
				responseXML, err := xml.Marshal(response)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal response: %w", err)
				}
				envelope.Body.Contents = responseXML
			}
		}
	}
	
	// Marshal the envelope
	var result []byte
	var err error
	
	if b.prettyPrint {
		result, err = xml.MarshalIndent(envelope, "", "\t")
	} else {
		result, err = xml.Marshal(envelope)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Add XML declaration
	xmlDeclaration := `<?xml version="1.0" encoding="UTF-8"?>`
	return append([]byte(xmlDeclaration), result...), nil
}

// BuildRPCRequest builds a TR069 RPC request.
// BuildRPCRequest 构建 TR069 RPC 请求。
func (b *Builder) BuildRPCRequest(ctx context.Context, method string, params map[string]interface{}) ([]byte, error) {
	// Create the RPC request structure
	request := struct {
		XMLName xml.Name
		Params  map[string]interface{} `xml:",omitempty"`
	}{
		XMLName: xml.Name{Local: "cwmp:" + method},
		Params:  params,
	}

	// Marshal the request body
	bodyContent, err := xml.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	// Create SOAP envelope
	envelope := struct {
		XMLName xml.Name `xml:"SOAP-ENV:Envelope"`
		SoapEnv string   `xml:"xmlns:SOAP-ENV,attr"`
		SoapEnc string   `xml:"xmlns:SOAP-ENC,attr"`
		XSD     string   `xml:"xmlns:xsd,attr"`
		XSI     string   `xml:"xmlns:xsi,attr"`
		CWMP    string   `xml:"xmlns:cwmp,attr"`
		Header  types.Header `xml:"SOAP-ENV:Header"`
		Body    struct {
			XMLName  xml.Name `xml:"SOAP-ENV:Body"`
			Contents []byte   `xml:",innerxml"`
		} `xml:"SOAP-ENV:Body"`
	}{
		SoapEnv: "http://schemas.xmlsoap.org/soap/envelope/",
		SoapEnc: "http://schemas.xmlsoap.org/soap/encoding/",
		XSD:     "http://www.w3.org/2001/XMLSchema",
		XSI:     "http://www.w3.org/2001/XMLSchema-instance",
		CWMP:    "urn:dslforum-org:cwmp-1-0",
	}

	envelope.Body.Contents = bodyContent

	return xml.Marshal(envelope)
}

// BuildRPCResponse builds a TR069 RPC response.
// BuildRPCResponse 构建 TR069 RPC 响应。
func (b *Builder) BuildRPCResponse(ctx context.Context, method string, params map[string]interface{}) ([]byte, error) {
	// Create response message
	msg := &interfaces.Message{
		Method: method,
	}
	
	// Convert params to Parameter slice
	var parameters []interfaces.Parameter
	for name, value := range params {
		parameters = append(parameters, interfaces.Parameter{
			Name:  name,
			Value: value,
			Type:  fmt.Sprintf("%T", value),
		})
	}
	msg.Parameters = parameters
	
	// Build the message
	return b.BuildMessage(ctx, msg)
}

// BuildFault builds a TR069 fault response.
// BuildFault 构建 TR069 故障响应。
func (b *Builder) BuildFault(ctx context.Context, code int, faultString string) ([]byte, error) {
	// Create fault message
	msg := &interfaces.Message{
		Fault: &interfaces.Fault{
			FaultCode:   code,
			FaultString: faultString,
		},
	}
	
	// Build the message
	return b.BuildMessage(ctx, msg)
}

// SetPrettyPrint sets whether the output should be formatted for readability.
// SetPrettyPrint 设置输出是否应格式化以提高可读性。
func (b *Builder) SetPrettyPrint(pretty bool) {
	b.prettyPrint = pretty
}

// GetPrettyPrint returns the current pretty print setting.
// GetPrettyPrint 返回当前的格式化输出设置。
func (b *Builder) GetPrettyPrint() bool {
	return b.prettyPrint
}

// SetSecurityConfig sets the security configuration for the builder.
// SetSecurityConfig 为构建器设置安全配置。
func (b *Builder) SetSecurityConfig(config *types.SecurityConfig) {
	b.securityConfig = config
}

// BuildCustomRPCResponse builds a response for a custom RPC method.
// BuildCustomRPCResponse 为自定义 RPC 方法构建响应。
func (b *Builder) BuildCustomRPCResponse(ctx context.Context, method string, result interface{}) ([]byte, error) {
	// Create response message
	msg := &interfaces.Message{
		Method: method,
	}
	
	// Convert result to Parameter slice if it's a map
	if params, ok := result.(map[string]interface{}); ok {
		var parameters []interfaces.Parameter
		for name, value := range params {
			parameters = append(parameters, interfaces.Parameter{
				Name:  name,
				Value: value,
				Type:  fmt.Sprintf("%T", value),
			})
		}
		msg.Parameters = parameters
	}
	
	// Build the message
	return b.BuildMessage(ctx, msg)
}
