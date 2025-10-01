// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

import (
	"context"
	"io"
)

// StreamParser is the interface for streaming TR069 message parsing.
// StreamParser 是流式 TR069 消息解析的接口。
type StreamParser interface {
	// ParseStream parses a TR069 message from a stream.
	// ParseStream 从流中解析 TR069 消息。
	ParseStream(ctx context.Context, reader io.Reader) (*Message, error)

	// ParseStreamWithCallback parses a TR069 message from a stream and calls the callback
	// for each parsed element.
	// ParseStreamWithCallback 从流中解析 TR069 消息，并为每个解析的元素调用回调函数。
	ParseStreamWithCallback(ctx context.Context, reader io.Reader, callback ParseCallback) error

	// SetBufferSize sets the buffer size for streaming parsing.
	// SetBufferSize 设置流式解析的缓冲区大小。
	SetBufferSize(size int)

	// GetBufferSize returns the current buffer size.
	// GetBufferSize 返回当前缓冲区大小。
	GetBufferSize() int

	// SetMemoryLimit sets the memory limit for parsing large messages.
	// SetMemoryLimit 设置解析大型消息的内存限制。
	SetMemoryLimit(limit int64)

	// GetMemoryLimit returns the current memory limit.
	// GetMemoryLimit 返回当前内存限制。
	GetMemoryLimit() int64

	// GetCurrentMemoryUsage returns the current memory usage.
	// GetCurrentMemoryUsage 返回当前内存使用量。
	GetCurrentMemoryUsage() int64

	// ResetMemoryUsage resets the memory usage counter.
	// ResetMemoryUsage 重置内存使用计数器。
	ResetMemoryUsage()

	// SetStrictMode sets the parser to strict mode, which will enforce stricter validation.
	// SetStrictMode 设置解析器的严格模式，启用更严格的验证。
	SetStrictMode(strict bool)

	// GetStrictMode returns the current strict mode setting.
	// GetStrictMode 返回当前的严格模式设置。
	GetStrictMode() bool
}

// ParseCallback is a function type for handling parsed elements during streaming parsing.
// ParseCallback 是在流式解析期间处理解析元素的函数类型。
type ParseCallback func(element *ParsedElement) error

// ParsedElement represents a parsed XML element.
// ParsedElement 表示一个解析的 XML 元素。
type ParsedElement struct {
	// Type is the type of the element.
	// Type 是元素的类型。
	Type ElementType

	// Name is the element name.
	// Name 是元素名称。
	Name string

	// Value is the element value.
	// Value 是元素值。
	Value interface{}

	// Attributes contains the element attributes.
	// Attributes 包含元素属性。
	Attributes map[string]string

	// Content is the text content of the element.
	// Content 是元素的文本内容。
	Content string

	// Children contains child elements.
	// Children 包含子元素。
	Children []*ParsedElement
}

// ElementType represents the type of a parsed element.
// ElementType 表示解析元素的类型。
type ElementType string

// Element type constants.
// 元素类型常量。
const (
	ElementTypeStart    ElementType = "start"
	ElementTypeEnd      ElementType = "end"
	ElementTypeCharData ElementType = "chardata"
	ElementHeader       ElementType = "header"
	ElementBody         ElementType = "body"
	ElementMethod       ElementType = "method"
	ElementParameter    ElementType = "parameter"
	ElementFault        ElementType = "fault"
	ElementDeviceID     ElementType = "deviceID"
	ElementEvent        ElementType = "event"
	ElementParameterList ElementType = "parameterList"
)