// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

// Parameter represents a TR069 parameter with its name, value and type.
// Parameter 表示一个 TR069 参数，包含其名称、值和类型。
type Parameter struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

// Attribute represents an XML attribute.
// Attribute 表示一个 XML 属性。
type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Message represents a TR069 message.
// Message 表示一个 TR069 消息。
type Message struct {
	Method         string      `json:"method"`
	ID             string      `json:"id,omitempty"`
	Parameters     []Parameter `json:"parameters,omitempty"`
	Fault          *Fault      `json:"fault,omitempty"`
	SessionID      string      `json:"sessionId,omitempty"`
	HoldRequests   bool        `json:"holdRequests,omitempty"`
	NoMoreRequests bool        `json:"noMoreRequests,omitempty"`

	// Inform specific fields
	DeviceID     *DeviceID `json:"deviceId,omitempty"`
	Events       []Event   `json:"events,omitempty"`
	MaxEnvelopes int       `json:"maxEnvelopes,omitempty"`
	CurrentTime  string    `json:"currentTime,omitempty"`
	RetryCount   int       `json:"retryCount,omitempty"`

	// Stream parsing fields
	Content       []string    `json:"content,omitempty"`
	EnvelopeStart bool        `json:"envelopeStart,omitempty"`
	EnvelopeEnd   bool        `json:"envelopeEnd,omitempty"`
	HasHeader     bool        `json:"hasHeader,omitempty"`
	HasBody       bool        `json:"hasBody,omitempty"`
	EnvelopeAttrs []Attribute `json:"envelopeAttrs,omitempty"`
	Params        map[string]interface{}
}

// DeviceID represents a device identifier.
// DeviceID 表示设备标识符。
type DeviceID struct {
	Manufacturer string `json:"manufacturer"`
	OUI          string `json:"oui"`
	ProductClass string `json:"productClass"`
	SerialNumber string `json:"serialNumber"`
}

// Event type is defined in event.go to avoid duplication

// Fault represents a TR069 fault response.
// Fault 表示一个 TR069 故障响应。
type Fault struct {
	FaultCode   int    `json:"faultCode"`
	FaultString string `json:"faultString"`
}

// Common TR069 RPC methods
// 常见的 TR069 RPC 方法
const (
	MethodInform                          = "cwmp:Inform"
	MethodGetRPCMethods                   = "cwmp:GetRPCMethods"
	MethodGetParameterValues              = "cwmp:GetParameterValues"
	MethodSetParameterValues              = "cwmp:SetParameterValues"
	MethodGetParameterNames               = "cwmp:GetParameterNames"
	MethodAddObject                       = "cwmp:AddObject"
	MethodDeleteObject                    = "cwmp:DeleteObject"
	MethodDownload                        = "cwmp:Download"
	MethodUpload                          = "cwmp:Upload"
	MethodReboot                          = "cwmp:Reboot"
	MethodFactoryReset                    = "cwmp:FactoryReset"
	MethodGetQueuedTransfers              = "cwmp:GetQueuedTransfers"
	MethodScheduleInform                  = "cwmp:ScheduleInform"
	MethodSetVouchers                     = "cwmp:SetVouchers"
	MethodGetOptions                      = "cwmp:GetOptions"
	MethodTransferComplete                = "cwmp:TransferComplete"
	MethodAutonomousTransferComplete      = "cwmp:AutonomousTransferComplete"
	MethodDUStateChangeComplete           = "cwmp:DUStateChangeComplete"
	MethodAutonomousDUStateChangeComplete = "cwmp:AutonomousDUStateChangeComplete"
)
