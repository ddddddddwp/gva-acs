// Package main demonstrates how to use the TR069 library.
package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// Example demonstrates basic usage of the TR069 library.
func Example() {
	// Example 1: Create a parser and parse a message
	fmt.Println("=== Parser Example ===")
	parser := factory.NewParser(
		interfaces.WithStrictMode(true),
		interfaces.WithMaxDepth(100),
	)

	// Sample TR-069 Inform message
	sampleMessage := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope
    xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
    <soap-env:Header>
        <cwmp:ID soap-env:mustUnderstand="1">12345</cwmp:ID>
    </soap-env:Header>
    <soap-env:Body>
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
    </soap-env:Body>
</soap-env:Envelope>`)

	// Parse the message
	msg, err := parser.ParseMessage(context.Background(), sampleMessage)
	if err != nil {
		log.Fatalf("Failed to parse message: %v", err)
	}

	fmt.Printf("Parsed message method: %s\n", msg.Method)
	fmt.Printf("Session ID: %s\n", msg.SessionID)
	fmt.Printf("Parameters count: %d\n", len(msg.Parameters))
	for _, param := range msg.Parameters {
		fmt.Printf("  - %s: %v (%s)\n", param.Name, param.Value, param.Type)
	}

	// Example 2: Create a builder and build a response
	fmt.Println("\n=== Builder Example ===")
	builder := factory.NewBuilder(
		interfaces.WithPrettyPrint(true),
	)

	// Build a response message
	responseMsg := &interfaces.Message{
		Method:    interfaces.MethodInform,
		SessionID: "12345",
		Parameters: []interfaces.Parameter{
			{
				Name:  "Device.DeviceInfo.Manufacturer",
				Value: "ExampleCorp",
				Type:  "string",
			},
			{
				Name:  "Device.DeviceInfo.ModelName",
				Value: "ExampleModel",
				Type:  "string",
			},
		},
	}

	response, err := builder.BuildMessage(context.Background(), responseMsg)
	if err != nil {
		log.Fatalf("Failed to build response: %v", err)
	}

	fmt.Println("Built response:")
	fmt.Println(string(response))

	// Example 3: Create a fault response
	fmt.Println("\n=== Fault Response Example ===")
	faultResponse, err := builder.BuildFault(context.Background(), 9001, "Invalid parameter")
	if err != nil {
		log.Fatalf("Failed to build fault response: %v", err)
	}

	fmt.Println("Fault response:")
	fmt.Println(string(faultResponse))
}
