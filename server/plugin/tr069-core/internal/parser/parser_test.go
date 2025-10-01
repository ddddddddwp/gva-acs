// Package parser implements the TR069 message parser.
package parser

import (
 "context"
 "testing"
)

func TestParser_ParseMessage(t *testing.T) {
 parser := New()
 
 // Sample TR-069 Inform message
 sampleMessage := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="urn:dslforum-org:cwmp-1-0" xmlns:cwmp="urn:dslforum-org:cwmp-1-0" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <Header>
        <ID>12345</ID>
    </Header>
    <Body>
        <cwmp:Inform>
            <DeviceId>
                <Manufacturer>ExampleCorp</Manufacturer>
                <OUI>123456</OUI>
                <ProductClass>ExampleProduct</ProductClass>
                <SerialNumber>123456789</SerialNumber>
            </DeviceId>
            <Event>
                <EventStruct>
                    <EventCode>0 BOOTSTRAP</EventCode>
                    <CommandKey></CommandKey>
                </EventStruct>
            </Event>
            <MaxEnvelopes>1</MaxEnvelopes>
            <CurrentTime>2023-01-01T00:00:00Z</CurrentTime>
            <RetryCount>0</RetryCount>
            <ParameterList>
                <ParameterValueStruct>
                    <Name>Device.DeviceInfo.Manufacturer</Name>
                    <Value xsi:type="xsd:string">ExampleCorp</Value>
                </ParameterValueStruct>
                <ParameterValueStruct>
                    <Name>Device.DeviceInfo.ModelName</Name>
                    <Value xsi:type="xsd:string">ExampleModel</Value>
                </ParameterValueStruct>
            </ParameterList>
        </cwmp:Inform>
    </Body>
</Envelope>`)
 
 msg, err := parser.ParseMessage(context.Background(), sampleMessage)
 if err != nil {
  t.Fatalf("Failed to parse message: %v", err)
 }
 
 if msg.Method != "Inform" {
  t.Errorf("Expected method %s, got %s", "Inform", msg.Method)
 }
 
 if len(msg.Parameters) != 2 {
  t.Errorf("Expected 2 parameters, got %d", len(msg.Parameters))
 }
}

func TestParser_ParseRPCMethod(t *testing.T) {
 parser := New()
 
 sampleMessage := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="urn:dslforum-org:cwmp-1-0">
    <Body>
        <cwmp:Inform>
            <DeviceId>
                <Manufacturer>ExampleCorp</Manufacturer>
            </DeviceId>
        </cwmp:Inform>
    </Body>
</Envelope>`)
 
 method, err := parser.ParseRPCMethod(context.Background(), sampleMessage)
 if err != nil {
  t.Fatalf("Failed to parse RPC method: %v", err)
 }
 
 if method != "Inform" {
  t.Errorf("Expected method %s, got %s", "Inform", method)
 }
}

func TestParser_SetStrictMode(t *testing.T) {
 parser := New()
 
 parser.SetStrictMode(true)
 if !parser.GetStrictMode() {
  t.Error("Expected strict mode to be true")
 }
 
 parser.SetStrictMode(false)
 if parser.GetStrictMode() {
  t.Error("Expected strict mode to be false")
 }
}