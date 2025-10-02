package tr069server

import (
	"encoding/xml"
	"time"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	SerialNumber  string    `json:"serial_number"`
	ProductClass  string    `json:"product_class"`
	Manufacturer  string    `json:"manufacturer"`
	OUI           string    `json:"oui"`
	RemoteAddr    string    `json:"remote_addr"`
	UserAgent     string    `json:"user_agent"`
	Timestamp     time.Time `json:"timestamp"`
}

// SOAP信封结构
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"soap:Envelope"`
	Header  *SOAPHeader `xml:"soap:Header,omitempty"`
	Body    *SOAPBody   `xml:"soap:Body"`
}

// SOAP头部
type SOAPHeader struct {
	ID string `xml:"cwmp:ID,omitempty"`
}

// SOAP主体
type SOAPBody struct {
	// CPE发送给ACS的消息
	Inform                      *Inform                      `xml:"cwmp:Inform,omitempty"`
	GetRPCMethodsResponse       *GetRPCMethodsResponse       `xml:"cwmp:GetRPCMethodsResponse,omitempty"`
	GetParameterValuesResponse  *GetParameterValuesResponse  `xml:"cwmp:GetParameterValuesResponse,omitempty"`
	SetParameterValuesResponse  *SetParameterValuesResponse  `xml:"cwmp:SetParameterValuesResponse,omitempty"`
	GetParameterNamesResponse   *GetParameterNamesResponse   `xml:"cwmp:GetParameterNamesResponse,omitempty"`
	GetParameterAttributesResponse *GetParameterAttributesResponse `xml:"cwmp:GetParameterAttributesResponse,omitempty"`
	SetParameterAttributesResponse *SetParameterAttributesResponse `xml:"cwmp:SetParameterAttributesResponse,omitempty"`
	AddObjectResponse           *AddObjectResponse           `xml:"cwmp:AddObjectResponse,omitempty"`
	DeleteObjectResponse        *DeleteObjectResponse        `xml:"cwmp:DeleteObjectResponse,omitempty"`
	DownloadResponse            *DownloadResponse            `xml:"cwmp:DownloadResponse,omitempty"`
	UploadResponse              *UploadResponse              `xml:"cwmp:UploadResponse,omitempty"`
	RebootResponse              *RebootResponse              `xml:"cwmp:RebootResponse,omitempty"`
	FactoryResetResponse        *FactoryResetResponse        `xml:"cwmp:FactoryResetResponse,omitempty"`
	
	// ACS发送给CPE的消息
	InformResponse              *InformResponse              `xml:"cwmp:InformResponse,omitempty"`
	GetRPCMethods               *GetRPCMethods               `xml:"cwmp:GetRPCMethods,omitempty"`
	GetParameterValues          *GetParameterValues          `xml:"cwmp:GetParameterValues,omitempty"`
	SetParameterValues          *SetParameterValues          `xml:"cwmp:SetParameterValues,omitempty"`
	GetParameterNames           *GetParameterNames           `xml:"cwmp:GetParameterNames,omitempty"`
	GetParameterAttributes      *GetParameterAttributes      `xml:"cwmp:GetParameterAttributes,omitempty"`
	SetParameterAttributes      *SetParameterAttributes      `xml:"cwmp:SetParameterAttributes,omitempty"`
	AddObject                   *AddObject                   `xml:"cwmp:AddObject,omitempty"`
	DeleteObject                *DeleteObject                `xml:"cwmp:DeleteObject,omitempty"`
	Download                    *Download                    `xml:"cwmp:Download,omitempty"`
	Upload                      *Upload                      `xml:"cwmp:Upload,omitempty"`
	Reboot                      *Reboot                      `xml:"cwmp:Reboot,omitempty"`
	FactoryReset                *FactoryReset                `xml:"cwmp:FactoryReset,omitempty"`
	
	// SOAP错误
	Fault                       *SOAPFault                   `xml:"soap:Fault,omitempty"`
}

// SOAP错误
type SOAPFault struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
	Detail      string `xml:"detail,omitempty"`
}

// 设备ID
type DeviceID struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

// 事件结构
type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

// 参数值对
type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
	Type  string `xml:"Type,attr"`
}

// 参数信息结构
type ParameterInfoStruct struct {
	Name     string `xml:"Name"`
	Writable bool   `xml:"Writable"`
}

// 参数属性结构
type ParameterAttributeStruct struct {
	Name         string `xml:"Name"`
	Notification int    `xml:"Notification"`
	AccessList   string `xml:"AccessList"`
}

// Inform消息
type Inform struct {
	DeviceId         DeviceID                   `xml:"DeviceId"`
	Event            []EventStruct              `xml:"Event>EventStruct"`
	MaxEnvelopes     int                        `xml:"MaxEnvelopes"`
	CurrentTime      string                     `xml:"CurrentTime"`
	RetryCount       int                        `xml:"RetryCount"`
	ParameterList    []ParameterValueStruct     `xml:"ParameterList>ParameterValueStruct"`
}

// InformResponse消息
type InformResponse struct {
	MaxEnvelopes int `xml:"MaxEnvelopes"`
}

// GetRPCMethods消息
type GetRPCMethods struct{}

// GetRPCMethodsResponse消息
type GetRPCMethodsResponse struct {
	MethodList []string `xml:"MethodList>string"`
}

// GetParameterValues消息
type GetParameterValues struct {
	ParameterNames []string `xml:"ParameterNames>string"`
}

// GetParameterValuesResponse消息
type GetParameterValuesResponse struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

// SetParameterValues消息
type SetParameterValues struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
	ParameterKey  string                 `xml:"ParameterKey"`
}

// SetParameterValuesResponse消息
type SetParameterValuesResponse struct {
	Status int `xml:"Status"`
}

// GetParameterNames消息
type GetParameterNames struct {
	ParameterPath string `xml:"ParameterPath"`
	NextLevel     bool   `xml:"NextLevel"`
}

// GetParameterNamesResponse消息
type GetParameterNamesResponse struct {
	ParameterList []ParameterInfoStruct `xml:"ParameterList>ParameterInfoStruct"`
}

// GetParameterAttributes消息
type GetParameterAttributes struct {
	ParameterNames []string `xml:"ParameterNames>string"`
}

// GetParameterAttributesResponse消息
type GetParameterAttributesResponse struct {
	ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct"`
}

// SetParameterAttributes消息
type SetParameterAttributes struct {
	ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct"`
}

// SetParameterAttributesResponse消息
type SetParameterAttributesResponse struct {
	Status int `xml:"Status"`
}

// AddObject消息
type AddObject struct {
	ObjectName   string `xml:"ObjectName"`
	ParameterKey string `xml:"ParameterKey"`
}

// AddObjectResponse消息
type AddObjectResponse struct {
	InstanceNumber int `xml:"InstanceNumber"`
	Status         int `xml:"Status"`
}

// DeleteObject消息
type DeleteObject struct {
	ObjectName   string `xml:"ObjectName"`
	ParameterKey string `xml:"ParameterKey"`
}

// DeleteObjectResponse消息
type DeleteObjectResponse struct {
	Status int `xml:"Status"`
}

// Download消息
type Download struct {
	CommandKey   string `xml:"CommandKey"`
	FileType     string `xml:"FileType"`
	URL          string `xml:"URL"`
	Username     string `xml:"Username"`
	Password     string `xml:"Password"`
	FileSize     int    `xml:"FileSize"`
	TargetFileName string `xml:"TargetFileName"`
	DelaySeconds int    `xml:"DelaySeconds"`
	SuccessURL   string `xml:"SuccessURL"`
	FailureURL   string `xml:"FailureURL"`
}

// DownloadResponse消息
type DownloadResponse struct {
	Status       int    `xml:"Status"`
	StartTime    string `xml:"StartTime"`
	CompleteTime string `xml:"CompleteTime"`
}

// Upload消息
type Upload struct {
	CommandKey   string `xml:"CommandKey"`
	FileType     string `xml:"FileType"`
	URL          string `xml:"URL"`
	Username     string `xml:"Username"`
	Password     string `xml:"Password"`
	DelaySeconds int    `xml:"DelaySeconds"`
}

// UploadResponse消息
type UploadResponse struct {
	Status       int    `xml:"Status"`
	StartTime    string `xml:"StartTime"`
	CompleteTime string `xml:"CompleteTime"`
}

// Reboot消息
type Reboot struct {
	CommandKey string `xml:"CommandKey"`
}

// RebootResponse消息
type RebootResponse struct{}

// FactoryReset消息
type FactoryReset struct{}

// FactoryResetResponse消息
type FactoryResetResponse struct{}

// CWMP错误代码
const (
	// 通用错误
	CWMP_OK                    = 0
	CWMP_METHOD_NOT_SUPPORTED  = 9000
	CWMP_REQUEST_DENIED        = 9001
	CWMP_INTERNAL_ERROR        = 9002
	CWMP_INVALID_ARGUMENTS     = 9003
	CWMP_RESOURCES_EXCEEDED    = 9004
	CWMP_INVALID_PARAMETER_NAME = 9005
	CWMP_INVALID_PARAMETER_TYPE = 9006
	CWMP_INVALID_PARAMETER_VALUE = 9007
	CWMP_ATTEMPT_TO_SET_NON_WRITABLE_PARAMETER = 9008
	CWMP_NOTIFICATION_REQUEST_REJECTED = 9009
	CWMP_DOWNLOAD_FAILURE      = 9010
	CWMP_UPLOAD_FAILURE        = 9011
	CWMP_FILE_TRANSFER_SERVER_AUTHENTICATION_FAILURE = 9012
	CWMP_UNSUPPORTED_PROTOCOL_FOR_FILE_TRANSFER = 9013
	CWMP_FILE_TRANSFER_UNABLE_TO_JOIN_MULTICAST_GROUP = 9014
	CWMP_FILE_TRANSFER_UNABLE_TO_CONTACT_FILE_SERVER = 9015
	CWMP_FILE_TRANSFER_UNABLE_TO_ACCESS_FILE = 9016
	CWMP_FILE_TRANSFER_UNABLE_TO_COMPLETE_DOWNLOAD = 9017
	CWMP_FILE_TRANSFER_FILE_CORRUPTED = 9018
	CWMP_FILE_TRANSFER_FILE_AUTHENTICATION_FAILURE = 9019
)

// CWMP事件代码
const (
	EVENT_BOOTSTRAP           = "0 BOOTSTRAP"
	EVENT_BOOT                = "1 BOOT"
	EVENT_PERIODIC            = "2 PERIODIC"
	EVENT_SCHEDULED           = "3 SCHEDULED"
	EVENT_VALUE_CHANGE        = "4 VALUE CHANGE"
	EVENT_KICKED              = "5 KICKED"
	EVENT_CONNECTION_REQUEST  = "6 CONNECTION REQUEST"
	EVENT_TRANSFER_COMPLETE   = "7 TRANSFER COMPLETE"
	EVENT_DIAGNOSTICS_COMPLETE = "8 DIAGNOSTICS COMPLETE"
	EVENT_REQUEST_DOWNLOAD    = "9 REQUEST DOWNLOAD"
	EVENT_AUTONOMOUS_TRANSFER_COMPLETE = "10 AUTONOMOUS TRANSFER COMPLETE"
	EVENT_DU_STATE_CHANGE_COMPLETE = "11 DU STATE CHANGE COMPLETE"
	EVENT_AUTONOMOUS_DU_STATE_CHANGE_COMPLETE = "12 AUTONOMOUS DU STATE CHANGE COMPLETE"
	EVENT_WAKEUP              = "13 WAKEUP"
)