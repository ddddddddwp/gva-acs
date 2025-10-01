package main

import (
 "context"
 "encoding/xml"
 "fmt"
 "github.com/root/demo/tr069/factory"
 "github.com/root/demo/tr069/interfaces"
 "github.com/root/demo/tr069/errors"
)

// DebugParserExample demonstrates how to use the parser with a sample message
func DebugParserExample() {
 // Sample TR-069 Inform message from test
 sampleMessage := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
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
                    <n>Device.DeviceInfo.Manufacturer</n>
                    <Value type="string">ExampleCorp</Value>
                </ParameterValueStruct>
                <ParameterValueStruct>
                    <n>Device.DeviceInfo.ModelName</n>
                    <Value type="string">ExampleModel</Value>
                </ParameterValueStruct>
            </ParameterList>
        </cwmp:Inform>
    </Body>
</Envelope>`)

 fmt.Println("Parsing envelope...")
 
 // 使用工厂创建解析器
 parser := factory.NewParser()
 
 ctx := context.Background()
 message, err := parser.ParseMessage(ctx, sampleMessage)
 if err != nil {
     fmt.Printf("Error parsing message: %v\n", err)
     return
 }
 
 fmt.Printf("Successfully parsed message with method: %s\n", message.Method)
 fmt.Printf("Parameters: %d\n", len(message.Parameters))
 for i, param := range message.Parameters {
     fmt.Printf("  %d. %s = %v (%s)\n", i+1, param.Name, param.Value, param.Type)
 }
 if err := xml.Unmarshal(sampleMessage, &interfaces.Envelope{}); err != nil {
   fmt.Printf("Failed to parse envelope: %v\n", err)
   return
  }

 fmt.Printf("Envelope parsed successfully\n")
 fmt.Printf("Header ID: %s\n", envelope.Header.ID)
 fmt.Printf("Body contents length: %d\n", len(envelope.Body.Contents))
 fmt.Printf("Body contents: %s\n", string(envelope.Body.Contents))

 fmt.Println("\nParsing RPC method...")
 parser := parser.New()
 method, err := parser.ParseRPCMethod(context.Background(), envelope.Body.Contents)
 if err != nil {
  fmt.Printf("Failed to parse RPC method: %v\n", err)
  return
 }

 fmt.Printf("RPC method: %s\n", method)

 fmt.Println("\nParsing specific method...")
 switch method {
 case "cwmp:Inform":
  inform := &types.Inform{}
  if err := xml.Unmarshal(envelope.Body.Contents, inform); err != nil {
   fmt.Printf("Failed to parse Inform: %v\n", err)
   // Let's print the actual error
   fmt.Printf("Error type: %T\n", err)
   if err == errors.ErrInvalidMessage {
    fmt.Println("This is our custom error")
   }
   return
  }

  fmt.Printf("Inform parsed successfully\n")
  fmt.Printf("DeviceId Manufacturer: %s\n", inform.DeviceId.Manufacturer)
  fmt.Printf("ParameterList length: %d\n", len(inform.ParameterList.Parameters))
 }
}