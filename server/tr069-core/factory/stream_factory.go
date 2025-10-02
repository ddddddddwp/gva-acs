// Package factory provides factory functions for creating TR069 components.
// 包 factory 提供创建 TR069 组件的工厂函数。
package factory

import (
	"context"
	"io"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/stream"
)

// StreamParserOption represents an option for configuring a stream parser.
// StreamParserOption 表示配置流式解析器的选项。
type StreamParserOption func(interfaces.StreamParser)

// WithBufferSize sets the buffer size for the stream parser.
// WithBufferSize 设置流式解析器的缓冲区大小。
func WithBufferSize(size int) StreamParserOption {
	return func(parser interfaces.StreamParser) {
		parser.SetBufferSize(size)
	}
}

// WithMemoryLimit sets the memory limit for the stream parser.
// WithMemoryLimit 设置流式解析器的内存限制。
func WithMemoryLimit(limit int64) StreamParserOption {
	return func(parser interfaces.StreamParser) {
		parser.SetMemoryLimit(limit)
	}
}

// WithStrictMode sets the strict mode for the stream parser.
// WithStrictMode 设置流式解析器的严格模式。
func WithStrictMode(strict bool) StreamParserOption {
	return func(parser interfaces.StreamParser) {
		parser.SetStrictMode(strict)
	}
}

// CreateStreamParser creates a new stream parser.
// CreateStreamParser 创建一个新的流式解析器。
func CreateStreamParser() interfaces.StreamParser {
	return stream.NewStreamParser()
}

// CreateStreamParserWithOptions creates a new stream parser with custom options.
// CreateStreamParserWithOptions 创建一个具有自定义选项的新流式解析器。
func CreateStreamParserWithOptions(bufferSize int, memoryLimit int64, strictMode bool) interfaces.StreamParser {
	parser := stream.NewStreamParser()
	parser.SetBufferSize(bufferSize)
	parser.SetMemoryLimit(memoryLimit)
	parser.SetStrictMode(strictMode)
	return parser
}

// CreateStreamParserWithFunctionalOptions creates a new stream parser with functional options.
// CreateStreamParserWithFunctionalOptions 使用函数式选项创建一个新的流式解析器。
func CreateStreamParserWithFunctionalOptions(options ...StreamParserOption) interfaces.StreamParser {
	parser := stream.NewStreamParser()
	for _, option := range options {
		option(parser)
	}
	return parser
}

// ParseWithTimeout parses a stream with a timeout.
// ParseWithTimeout 使用超时解析流。
func ParseWithTimeout(parser interfaces.StreamParser, reader io.Reader, timeout time.Duration) (*interfaces.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return parser.ParseStream(ctx, reader)
}

// ParseWithCallback parses a stream and calls the callback for each parsed element.
// ParseWithCallback 解析流并为每个解析的元素调用回调函数。
func ParseWithCallback(parser interfaces.StreamParser, reader io.Reader, callback interfaces.ParseCallback) error {
	return parser.ParseStreamWithCallback(context.Background(), reader, callback)
}
