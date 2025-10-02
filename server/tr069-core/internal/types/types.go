// Package types defines internal data types for the TR069 library.
package types

import (
 "encoding/xml"
 
 "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// Envelope represents a SOAP envelope.
type Envelope struct {
 XMLName xml.Name `xml:"Envelope"`
 Header  Header   `xml:"Header"`
 Body    Body     `xml:"Body"`
}

// Header represents a SOAP header.
type Header struct {
 XMLName        xml.Name `xml:"Header"`
 ID             string   `xml:"ID"`
 SessionID      string   `xml:"SessionID,omitempty"`
 HoldRequests   bool     `xml:"HoldRequests,omitempty"`
 NoMoreRequests int      `xml:"NoMoreRequests,omitempty"`
}

// Body represents a SOAP body.
type Body struct {
 XMLName  xml.Name   `xml:"Body"`
 Contents []byte     `xml:",innerxml"`
}

// Inform represents a TR-069 Inform RPC method.
type Inform struct {
 XMLName xml.Name `xml:"Inform"`
 
 DeviceId      DeviceId           `xml:"DeviceId"`
 Event         EventList          `xml:"Event"`
 MaxEnvelopes  int                `xml:"MaxEnvelopes"`
 CurrentTime string `xml:"CurrentTime"`
 RetryCount    int                `xml:"RetryCount"`
 ParameterList ParameterValueList `xml:"ParameterList"`
}

// DeviceId represents a device identifier.
type DeviceId struct {
 Manufacturer string `xml:"Manufacturer"`
 OUI          string `xml:"OUI"`
 ProductClass string `xml:"ProductClass"`
 SerialNumber string `xml:"SerialNumber"`
}

// EventList represents a list of events.
type EventList struct {
 XMLName xml.Name `xml:"Event"`
 ArrayType string     `xml:"arrayType,attr"`
 Events []EventStruct `xml:"EventStruct"`
}

// EventStruct represents an event structure
type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

// ParameterValueList represents a list of parameter values.
type ParameterValueList struct {
 XMLName xml.Name `xml:"ParameterList"`
 ArrayType string                  `xml:"arrayType,attr"`
 Parameters []ParameterValueStruct `xml:"ParameterValueStruct"`
}

// ParameterValueStruct represents a parameter value structure
type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value Value  `xml:"Value"`
}

// Value represents a parameter value with type information.
type Value struct {
 Type  string `xml:"type,attr"`
 Value string `xml:",chardata"`
}

// InformResponse represents a TR-069 InformResponse RPC method.
type InformResponse struct {
 XMLName xml.Name `xml:"InformResponse"`
 MaxEnvelopes int `xml:"MaxEnvelopes"`
}

// GetParameterValues represents a TR-069 GetParameterValues RPC method.
type GetParameterValues struct {
 XMLName        xml.Name       `xml:"GetParameterValues"`
 ParameterNames ParameterNames `xml:"ParameterNames"`
}

// ParameterNames represents a list of parameter names.
type ParameterNames struct {
 Names []string `xml:"string"`
}

// SetParameterValues represents a TR-069 SetParameterValues RPC method.
type SetParameterValues struct {
 XMLName       xml.Name           `xml:"SetParameterValues"`
 ParameterList ParameterValueList `xml:"ParameterList"`
 ParameterKey  string             `xml:"ParameterKey"`
}

// GetParameterNames represents a TR-069 GetParameterNames RPC method.
type GetParameterNames struct {
 XMLName xml.Name `xml:"GetParameterNames"`
 ParameterPath string `xml:"ParameterPath"`
 NextLevel bool `xml:"NextLevel"`
}

// AddObject represents a TR-069 AddObject RPC method.
type AddObject struct {
 XMLName xml.Name `xml:"AddObject"`
 ObjectName string `xml:"ObjectName"`
 ParameterKey string `xml:"ParameterKey"`
}

// DeleteObject represents a TR-069 DeleteObject RPC method.
type DeleteObject struct {
 XMLName xml.Name `xml:"DeleteObject"`
 ObjectName string `xml:"ObjectName"`
 ParameterKey string `xml:"ParameterKey"`
}

// Download represents a TR-069 Download RPC method.
type Download struct {
 XMLName xml.Name `xml:"Download"`
 CommandKey string `xml:"CommandKey"`
 FileType string `xml:"FileType"`
 URL string `xml:"URL"`
 Username string `xml:"Username"`
 Password string `xml:"Password"`
 FileSize int `xml:"FileSize"`
 TargetFileName string `xml:"TargetFileName"`
 DelaySeconds int `xml:"DelaySeconds"`
 SuccessURL string `xml:"SuccessURL"`
 FailureURL string `xml:"FailureURL"`
}

// Upload represents a TR-069 Upload RPC method.
type Upload struct {
 XMLName xml.Name `xml:"Upload"`
 CommandKey string `xml:"CommandKey"`
 FileType string `xml:"FileType"`
 URL string `xml:"URL"`
 Username string `xml:"Username"`
 Password string `xml:"Password"`
 DelaySeconds int `xml:"DelaySeconds"`
}

// Reboot represents a TR-069 Reboot RPC method.
type Reboot struct {
 XMLName xml.Name `xml:"Reboot"`
 CommandKey string `xml:"CommandKey"`
}

// FactoryReset represents a TR-069 FactoryReset RPC method.
type FactoryReset struct {
 XMLName xml.Name `xml:"FactoryReset"`
}

// Fault represents a TR-069 Fault response.
type Fault struct {
 XMLName xml.Name `xml:"Fault"`
 FaultCode int `xml:"FaultCode"`
 FaultString string `xml:"FaultString"`
}

// ToInterface converts internal types to interface types
func (p *ParameterValueStruct) ToInterface() interfaces.Parameter {
 return interfaces.Parameter{
  Name:  p.Name,
  Value: p.Value.Value,
  Type:  p.Value.Type,
 }
}