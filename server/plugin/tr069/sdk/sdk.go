package sdk

import (
	"encoding/xml"
	"fmt"
)

// Mock SDK Implementation

// Packet represents a parsed TR069 packet (Simplified)
type Packet struct {
	ID          string `xml:"Header>ID"`
	Method      string `xml:"Body>Method"` // e.g., Inform, GetParameterValues
	DeviceID    string `xml:"Body>Inform>DeviceId>SerialNumber"`
	EventCode   string `xml:"Body>Inform>Event>EventStruct>EventCode"`
	RawData     []byte
}

// Response represents a response to be sent back
type Response struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	Body    struct {
		Content string `xml:",innerxml"`
	} `xml:"soap:Body"`
}

// Parse parses the raw XML data
func Parse(data []byte) (*Packet, error) {
	// In a real SDK, this would do complex XML parsing
	// Here we just do a simple mock
	fmt.Printf("[SDK] Parsing data: %s\n", string(data))
	
	return &Packet{
		ID:        "12345",
		Method:    "Inform",
		DeviceID:  "MOCK_DEVICE_001",
		EventCode: "2 PERIODIC",
		RawData:   data,
	}, nil
}

// BuildResponse constructs a CWMP response
func BuildResponse(packet *Packet) []byte {
	return []byte(fmt.Sprintf(`<soap:Envelope><soap:Body><cwmp:InformResponse><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse></soap:Body></soap:Envelope>`))
}
