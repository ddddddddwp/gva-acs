// Package stream implements the TR069 stream parser for handling large XML messages.
// 包 stream 实现了 TR069 流式解析器，用于处理大型 XML 消息。
package stream

import (
	"strings"
	"testing"
)

// TestStreamParserIntegration tests the complete stream parsing flow.
// TestStreamParserIntegration 测试完整的流解析流程。
func TestStreamParserIntegration(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024) // 1KB buffer, 1MB max memory
	
	// Create a test XML input that simulates a real TR069 message
	// 创建模拟真实 TR069 消息的测试 XML 输入
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
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
                <OUI>123456</OUI>
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
            <ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[3]">
                <ParameterValueStruct>
                    <Name>InternetGatewayDevice.DeviceInfo.SpecVersion</Name>
                    <Value xsi:type="xsd:string">1.0</Value>
                </ParameterValueStruct>
                <ParameterValueStruct>
                    <Name>InternetGatewayDevice.DeviceInfo.HardwareVersion</Name>
                    <Value xsi:type="xsd:string">1.0</Value>
                </ParameterValueStruct>
                <ParameterValueStruct>
                    <Name>InternetGatewayDevice.DeviceInfo.SoftwareVersion</Name>
                    <Value xsi:type="xsd:string">1.0</Value>
                </ParameterValueStruct>
            </ParameterList>
        </cwmp:Inform>
    </soap-env:Body>
</soap-env:Envelope>`
	
	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	elements, err := parser.ParseStream(reader)
	
	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}
	
	// Check that we got some elements
	// 检查是否获得了一些元素
	if len(elements) == 0 {
		t.Error("Expected to parse some elements, got none")
	}
	
	// Verify specific elements were parsed
	// 验证特定元素是否已解析
	// This would require a more detailed verification based on the parser implementation
	// 这需要基于解析器实现进行更详细的验证
	// For now, we'll just check that we have a reasonable number of elements
	// 现在，我们只检查元素数量是否合理
	if len(elements) < 5 {
		t.Errorf("Expected at least 5 elements, got %d", len(elements))
	}
}

// TestStreamParserWithCallbackIntegration tests stream parsing with callback.
// TestStreamParserWithCallbackIntegration 测试使用回调的流解析。
func TestStreamParserWithCallbackIntegration(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024) // 1KB buffer, 1MB max memory
	
	// Create a test XML input
	// 创建测试 XML 输入
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<root>
    <element1>value1</element1>
    <element2>value2</element2>
    <element3>value3</element3>
</root>`
	
	// Counter for processed elements
	// 已处理元素的计数器
	count := 0
	
	// Define a callback function
	// 定义回调函数
	callback := func(elementName string, elementValue string) error {
		count++
		// Verify that we're getting expected values
		// 验证是否获得预期值
		if elementName == "element1" && elementValue != "value1" {
			t.Errorf("Expected value1 for element1, got %s", elementValue)
		}
		if elementName == "element2" && elementValue != "value2" {
			t.Errorf("Expected value2 for element2, got %s", elementValue)
		}
		if elementName == "element3" && elementValue != "value3" {
			t.Errorf("Expected value3 for element3, got %s", elementValue)
		}
		return nil
	}
	
	// Parse the stream with callback
	// 使用回调解析流
	reader := strings.NewReader(xmlInput)
	err := parser.ParseStreamWithCallback(reader, callback)
	
	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStreamWithCallback failed: %v", err)
	}
	
	// Check that all elements were processed
	// 检查所有元素是否已处理
	expectedCount := 3
	if count != expectedCount {
		t.Errorf("Expected %d elements processed, got %d", expectedCount, count)
	}
}

// TestStreamParserLargeMessageIntegration tests parsing of a large TR069 message.
// TestStreamParserLargeMessageIntegration 测试解析大型 TR069 消息。
func TestStreamParserLargeMessageIntegration(t *testing.T) {
	// Create a stream parser with appropriate buffer settings
	// 创建具有适当缓冲区设置的流式解析器
	parser := NewStreamParser(8192, 10*1024*1024) // 8KB buffer, 10MB max memory
	
	// Create a large XML input simulating a parameter list with many entries
	// 创建模拟包含许多条目的参数列表的大型 XML 输入
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
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
                <OUI>123456</OUI>
                <ProductClass>ExampleModel</ProductClass>
                <SerialNumber>1234567890</SerialNumber>
            </DeviceId>
            <Event soap-enc:arrayType="cwmp:EventStruct[1]">
                <EventStruct>
                    <EventCode>2 PERIODIC</EventCode>
                    <CommandKey></CommandKey>
                </EventStruct>
            </Event>
            <MaxEnvelopes>1</MaxEnvelopes>
            <CurrentTime>2023-01-01T00:00:00Z</CurrentTime>
            <RetryCount>0</RetryCount>
            <ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[1000]">`)
	
	// Add many parameter value structs
	// 添加许多参数值结构
	for i := 0; i < 1000; i++ {
		xmlBuilder.WriteString(`<ParameterValueStruct>
            <Name>InternetGatewayDevice.DeviceInfo.Parameter`)
		xmlBuilder.WriteString(string(rune(i+'0')))
		xmlBuilder.WriteString(`</Name>
            <Value xsi:type="xsd:string">Value`)
		xmlBuilder.WriteString(string(rune(i+'0')))
		xmlBuilder.WriteString(`</Value>
        </ParameterValueStruct>`)
	}
	
	xmlBuilder.WriteString(`</ParameterList>
        </cwmp:Inform>
    </soap-env:Body>
</soap-env:Envelope>`)
	
	xmlInput := xmlBuilder.String()
	
	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	elements, err := parser.ParseStream(reader)
	
	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}
	
	// Check that we got elements
	// 检查是否获得元素
	if len(elements) == 0 {
		t.Error("Expected to parse some elements, got none")
	}
}

// TestStreamParserMemoryManagementIntegration tests memory management during parsing.
// TestStreamParserMemoryManagementIntegration 测试解析过程中的内存管理。
func TestStreamParserMemoryManagementIntegration(t *testing.T) {
	// Create a stream parser with limited memory
	// 创建具有有限内存的流式解析器
	parser := NewStreamParser(512, 2048) // 512B buffer, 2KB max memory
	
	// Create a moderately sized XML input
	// 创建中等大小的 XML 输入
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<root>`
	
	// Add a number of elements
	// 添加一些元素
	for i := 0; i < 50; i++ {
		xmlInput += "<element" + string(rune(i+'0')) + ">value" + string(rune(i+'0')) + "</element" + string(rune(i+'0')) + ">"
	}
	
	xmlInput += `</root>`
	
	// Check initial memory usage
	// 检查初始内存使用情况
	initialMemory := parser.GetCurrentMemoryUsage()
	if initialMemory != 0 {
		t.Errorf("Expected initial memory usage 0, got %d", initialMemory)
	}
	
	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	_, err := parser.ParseStream(reader)
	
	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}
	
	// Check final memory usage (should be 0 after parsing completes)
	// 检查最终内存使用情况（解析完成后应为 0）
	finalMemory := parser.GetCurrentMemoryUsage()
	if finalMemory != 0 {
		t.Errorf("Expected final memory usage 0, got %d", finalMemory)
	}
}

// TestStreamParserErrorHandlingIntegration tests error handling during parsing.
// TestStreamParserErrorHandlingIntegration 测试解析过程中的错误处理。
func TestStreamParserErrorHandlingIntegration(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024)
	
	// Create invalid XML input
	// 创建无效 XML 输入
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<root>
    <element1>value1</element1>
    <element2>value2</element2>
    <!-- Missing closing tag -->
`
	
	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	_, err := parser.ParseStream(reader)
	
	// Check that we got an error
	// 检查是否出现错误
	if err == nil {
		t.Error("Expected error for invalid XML, but got none")
	}
}