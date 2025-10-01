package main

import (
	"context"
	"fmt"
	"log"

	"github.com/root/demo/tr069/factory"
)

func main() {
	// 创建解析器实例
	p := factory.NewParser()
	
	// 创建构建器实例
	b := factory.NewBuilder()
	
	// 示例TR-069 Inform消息
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<soap-env:Envelope
		xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
		xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
		xmlns:xsd="http://www.w3.org/2001/XMLSchema"
		xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
		xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
		<soap-env:Header>
			<cwmp:ID soap-env:mustUnderstand="1">1234567890</cwmp:ID>
		</soap-env:Header>
		<soap-env:Body>
			<cwmp:Inform>
				<DeviceId>
					<Manufacturer>ExampleCorp</Manufacturer>
					<OUI>001122</OUI>
					<ProductClass>ExampleModel</ProductClass>
					<SerialNumber>1234567890</SerialNumber>
				</DeviceId>
				<Event soap-enc:arrayType="cwmp:EventStruct[1]">
					<EventStruct>
						<EventCode>0 BOOTSTRAP</EventCode>
						<CommandKey></CommandKey>
					</EventStruct>
				</Event>
				<MaxEnvelopes>1</MaxEnvelopes>
				<CurrentTime>2023-01-01T00:00:00Z</CurrentTime>
				<RetryCount>0</RetryCount>
				<ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[2]">
					<ParameterValueStruct>
						<Name>Device.DeviceInfo.Manufacturer</Name>
						<Value xsi:type="xsd:string">ExampleCorp</Value>
					</ParameterValueStruct>
					<ParameterValueStruct>
						<Name>Device.DeviceInfo.ModelName</Name>
						<Value xsi:type="xsd:string">Model-123</Value>
					</ParameterValueStruct>
				</ParameterList>
			</cwmp:Inform>
		</soap-env:Body>
	</soap-env:Envelope>`)
	
	// 解析消息
	msg, err := p.ParseMessage(context.Background(), xmlData)
	if err != nil {
		log.Fatalf("Failed to parse message: %v", err)
	}
	
	fmt.Printf("Parsed message method: %s\n", msg.Method)
	fmt.Printf("Number of parameters: %d\n", len(msg.Parameters))
	
	// 打印参数
	for _, param := range msg.Parameters {
		fmt.Printf("Parameter: %s = %v (%s)\n", param.Name, param.Value, param.Type)
	}
	
	// 构建响应消息
	responseData, err := b.BuildMessage(context.Background(), msg)
	if err != nil {
		log.Fatalf("Failed to build response: %v", err)
	}
	
	fmt.Printf("Response XML:\n%s\n", responseData)
	
	// 构建故障响应示例
	faultData, err := b.BuildFault(context.Background(), 9001, "Invalid parameter")
	if err != nil {
		log.Fatalf("Failed to build fault: %v", err)
	}
	
	fmt.Printf("Fault XML:\n%s\n", faultData)
}