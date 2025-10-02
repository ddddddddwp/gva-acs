// Package stream implements the TR069 stream parser for handling large XML messages.
// 包 stream 实现了 TR069 流式解析器，用于处理大型 XML 消息。
package stream

import (
	"strings"
	"testing"
)

// TestStreamParser_ParseStream tests the ParseStream method of the stream parser.
// TestStreamParser_ParseStream 测试流式解析器的 ParseStream 方法。
func TestStreamParser_ParseStream(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024) // 1KB buffer, 1MB max memory

	// Create a test XML input
	// 创建测试 XML 输入
	xmlInput := `<root>
	<element1>value1</element1>
	<element2>value2</element2>
	<element3>value3</element3>
</root>`

	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	elements, err := parser.ParseStream(reader)

	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}

	// Check that we got the expected elements
	// 检查是否获得了预期的元素
	expectedElements := 3
	if len(elements) != expectedElements {
		t.Errorf("Expected %d elements, got %d", expectedElements, len(elements))
	}
}

// TestStreamParser_ParseStreamWithCallback tests the ParseStreamWithCallback method of the stream parser.
// TestStreamParser_ParseStreamWithCallback 测试流式解析器的 ParseStreamWithCallback 方法。
func TestStreamParser_ParseStreamWithCallback(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024) // 1KB buffer, 1MB max memory

	// Create a test XML input
	// 创建测试 XML 输入
	xmlInput := `<root>
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
	// 检查所有元素都已处理
	expectedCount := 3
	if count != expectedCount {
		t.Errorf("Expected %d elements processed, got %d", expectedCount, count)
	}
}

// TestStreamParser_ParseLargeStream tests parsing of a large stream.
// TestStreamParser_ParseLargeStream 测试解析大型流。
func TestStreamParser_ParseLargeStream(t *testing.T) {
	// Create a stream parser with small buffer to test memory management
	// 创建具有小缓冲区的流式解析器以测试内存管理
	parser := NewStreamParser(128, 1024) // 128B buffer, 1KB max memory

	// Create a large XML input
	// 创建大型 XML 输入
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<root>\n")
	for i := 0; i < 100; i++ {
		xmlBuilder.WriteString("<element>")
		xmlBuilder.WriteString(strings.Repeat("a", 50)) // 50 character value
		xmlBuilder.WriteString("</element>\n")
	}
	xmlBuilder.WriteString("</root>")

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

	// Check that we got the expected elements
	// 检查是否获得了预期的元素
	expectedElements := 100
	if len(elements) != expectedElements {
		t.Errorf("Expected %d elements, got %d", expectedElements, len(elements))
	}
}

// TestStreamParser_ParseStreamWithMemoryLimit tests parsing with memory limit exceeded.
// TestStreamParser_ParseStreamWithMemoryLimit 测试超出内存限制的解析。
func TestStreamParser_ParseStreamWithMemoryLimit(t *testing.T) {
	// Create a stream parser with very small memory limit
	// 创建具有非常小内存限制的流式解析器
	parser := NewStreamParser(64, 128) // 64B buffer, 128B max memory

	// Create a large XML input that will exceed memory limit
	// 创建将超出内存限制的大型 XML 输入
	xmlInput := "<root>" + strings.Repeat("<element>value</element>", 100) + "</root>"

	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	_, err := parser.ParseStream(reader)

	// Check that we got a memory limit error
	// 检查是否出现内存限制错误
	if err == nil {
		t.Error("Expected memory limit error, but got none")
	}

	// Check that the error is related to memory limit
	// 检查错误是否与内存限制相关
	if !strings.Contains(err.Error(), "memory") {
		t.Errorf("Expected memory limit error, got: %v", err)
	}
}

// TestStreamParser_ParseStreamInvalidXML tests parsing of invalid XML.
// TestStreamParser_ParseStreamInvalidXML 测试解析无效 XML。
func TestStreamParser_ParseStreamInvalidXML(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024)

	// Create invalid XML input
	// 创建无效 XML 输入
	xmlInput := `<root>
	<element1>value1</element1>
	<element2>value2</element2>
	<element3>value3</element3>
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

// TestStreamParser_ParseStreamEmpty tests parsing of empty stream.
// TestStreamParser_ParseStreamEmpty 测试解析空流。
func TestStreamParser_ParseStreamEmpty(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024)

	// Parse an empty stream
	// 解析空流
	reader := strings.NewReader("")
	elements, err := parser.ParseStream(reader)

	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}

	// Check that we got no elements
	// 检查是否没有元素
	if len(elements) != 0 {
		t.Errorf("Expected 0 elements, got %d", len(elements))
	}
}

// TestStreamParser_ParseStreamWithNestedElements tests parsing of nested elements.
// TestStreamParser_ParseStreamWithNestedElements 测试解析嵌套元素。
func TestStreamParser_ParseStreamWithNestedElements(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(1024, 1024*1024)

	// Create XML with nested elements
	// 创建具有嵌套元素的 XML
	xmlInput := `<root>
	<parent>
		<child1>value1</child1>
		<child2>value2</child2>
	</parent>
	<parent>
		<child3>value3</child3>
		<child4>value4</child4>
	</parent>
</root>`

	// Parse the stream
	// 解析流
	reader := strings.NewReader(xmlInput)
	elements, err := parser.ParseStream(reader)

	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}

	// Check that we got the expected elements
	// 检查是否获得了预期的元素
	// Note: The parser should handle nested elements appropriately
	// 注意：解析器应适当地处理嵌套元素
	expectedElements := 2 // We expect 2 parent elements
	if len(elements) != expectedElements {
		t.Errorf("Expected %d elements, got %d", expectedElements, len(elements))
	}
}

// TestStreamParser_BufferManagement tests buffer management of the stream parser.
// TestStreamParser_BufferManagement 测试流式解析器的缓冲区管理。
func TestStreamParser_BufferManagement(t *testing.T) {
	// Create a stream parser
	// 创建流式解析器
	parser := NewStreamParser(256, 1024) // 256B buffer, 1KB max memory

	// Check initial buffer size
	// 检查初始缓冲区大小
	if parser.GetBufferSize() != 256 {
		t.Errorf("Expected buffer size 256, got %d", parser.GetBufferSize())
	}

	// Check initial memory usage
	// 检查初始内存使用情况
	if parser.GetCurrentMemoryUsage() != 0 {
		t.Errorf("Expected initial memory usage 0, got %d", parser.GetCurrentMemoryUsage())
	}

	// Parse a small stream
	// 解析小流
	xmlInput := "<root><element>value</element></root>"
	reader := strings.NewReader(xmlInput)
	_, err := parser.ParseStream(reader)

	// Check for errors
	// 检查错误
	if err != nil {
		t.Errorf("ParseStream failed: %v", err)
	}

	// Check final memory usage (should be 0 after parsing completes)
	// 检查最终内存使用情况（解析完成后应为 0）
	if parser.GetCurrentMemoryUsage() != 0 {
		t.Errorf("Expected final memory usage 0, got %d", parser.GetCurrentMemoryUsage())
	}
}
