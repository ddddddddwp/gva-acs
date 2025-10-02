package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// FullTR069Example 演示TR069库的完整使用流程
func FullTR069Example() {
	// 创建解析器和构建器实例
	parser := factory.NewParser()
	builder := factory.NewBuilder()

	// 示例TR069消息数据
	messageData := []byte(`
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
        </soap-env:Envelope>
    `)

	// 解析消息
	message, err := parser.ParseMessage(context.Background(), messageData)
	if err != nil {
		log.Fatalf("解析消息失败: %v", err)
	}

	fmt.Printf("解析的RPC方法: %s\n", message.Method)
	fmt.Printf("参数数量: %d\n", len(message.Parameters))

	// 打印参数
	for _, param := range message.Parameters {
		fmt.Printf("参数: %s = %v (%s)\n", param.Name, param.Value, param.Type)
	}

	// 构建响应消息
	responseMsg := &interfaces.Message{
		Method: "InformResponse",
		Parameters: []interfaces.Parameter{
			{
				Name:  "MaxEnvelopes",
				Value: 1,
				Type:  "int",
			},
		},
	}

	response, err := builder.BuildMessage(context.Background(), responseMsg)
	if err != nil {
		log.Fatalf("构建响应失败: %v", err)
	}

	fmt.Printf("构建的响应消息:\n%s\n", string(response))
}

// 示例函数调用
func ExampleFullTR069Example() {
	FullTR069Example()
}
