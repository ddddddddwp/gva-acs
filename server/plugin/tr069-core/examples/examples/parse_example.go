package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/root/demo/tr069/factory"
)

// ParseTR069Message 演示如何解析TR069 XML消息
func ParseTR069Message() {
	// 创建解析器实例
	parser := factory.NewParser()

	// 示例TR069 XML消息
	xmlMessage := `
	<soap-env:Envelope
		xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
		xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
		xmlns:xsd="http://www.w3.org/2001/XMLSchema"
		xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
		xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
		<soap-env:Header>
			<cwmp:HoldRequests soap-env:mustUnderstand="1">1</cwmp:HoldRequests>
		</soap-env:Header>
		<soap-env:Body>
			<cwmp:Inform>
				<DeviceId>
					<Manufacturer>ExampleCorp</Manufacturer>
					<OUI>001122</OUI>
					<ProductClass>Device</ProductClass>
					<SerialNumber>SN123456</SerialNumber>
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
	</soap-env:Envelope>`

	// 解析XML消息
	message, err := parser.ParseMessage(context.Background(), []byte(xmlMessage))
	if err != nil {
		log.Fatalf("解析消息失败: %v", err)
	}

	// 输出解析结果
	fmt.Printf("RPC方法: %s\n", message.Method)
	fmt.Printf("参数数量: %d\n", len(message.Parameters))

	// 输出参数详情
	for _, param := range message.Parameters {
		fmt.Printf("参数名: %s, 值: %v, 类型: %s\n", param.Name, param.Value, param.Type)
	}
}

// 示例函数调用
func ExampleParseTR069Message() {
	ParseTR069Message()
}