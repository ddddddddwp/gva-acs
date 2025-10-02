// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

import (
	"context"
)

// Version represents a TR-069 protocol version.
type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
}

// Parser is the interface for TR069 message parsing.
// Parser 是 TR069 消息解析的接口。
type Parser interface {
	// ParseMessage parses a TR069 message from raw bytes.
	// ParseMessage 从原始字节数据中解析 TR069 消息。
	ParseMessage(ctx context.Context, data []byte) (*Message, error)

	// ParseRPCMethod extracts the RPC method from the raw message.
	// ParseRPCMethod 从原始消息中提取 RPC 方法。
	ParseRPCMethod(ctx context.Context, data []byte) (string, error)

	// ParseParameterList extracts the parameter list from the raw message.
	// ParseParameterList 从原始消息中提取参数列表。
	ParseParameterList(ctx context.Context, data []byte) ([]Parameter, error)

	// SetStrictMode sets the parser to strict mode, which will enforce stricter validation.
	// SetStrictMode 设置解析器的严格模式，启用更严格的验证。
	SetStrictMode(strict bool)

	// GetStrictMode returns the current strict mode setting.
	// GetStrictMode 返回当前的严格模式设置。
	GetStrictMode() bool

	// DetectVersion detects the TR-069 protocol version from the message.
	// DetectVersion 从消息中检测 TR-069 协议版本。
	DetectVersion(data []byte) Version

	// IsVersionSupported checks if the given version is supported.
	// IsVersionSupported 检查给定版本是否受支持。
	IsVersionSupported(version Version) bool

	// RegisterCustomMethod registers a custom RPC method handler.
	// RegisterCustomMethod 注册自定义 RPC 方法处理器。
	RegisterCustomMethod(method string, handler func(ctx context.Context, params map[string]interface{}) (interface{}, error)) error

	// UnregisterCustomMethod unregisters a custom RPC method handler.
	// UnregisterCustomMethod 取消注册自定义 RPC 方法处理器。
	UnregisterCustomMethod(method string) error

	// ListCustomMethods returns a list of registered custom methods.
	// ListCustomMethods 返回已注册的自定义方法列表。
	ListCustomMethods() []string

	// IsCustomMethodRegistered checks if a custom method is registered.
	// IsCustomMethodRegistered 检查自定义方法是否已注册。
	IsCustomMethodRegistered(method string) bool
}
