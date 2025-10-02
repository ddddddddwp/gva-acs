// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

import (
	"context"
)

// Builder is the interface for TR069 message building.
// Builder 是 TR069 消息构建的接口。
type Builder interface {
	// BuildMessage builds a TR069 message from a Message struct.
	// BuildMessage 从 Message 结构体构建 TR069 消息。
	BuildMessage(ctx context.Context, msg *Message) ([]byte, error)

	// BuildRPCRequest builds a TR069 RPC request.
	// BuildRPCRequest 构建 TR069 RPC 请求。
	BuildRPCRequest(ctx context.Context, method string, params map[string]interface{}) ([]byte, error)

	// BuildRPCResponse builds a TR069 RPC response.
	// BuildRPCResponse 构建 TR069 RPC 响应。
	BuildRPCResponse(ctx context.Context, method string, params map[string]interface{}) ([]byte, error)

	// BuildFault builds a TR069 fault response.
	// BuildFault 构建 TR069 故障响应。
	BuildFault(ctx context.Context, code int, faultString string) ([]byte, error)

	// SetPrettyPrint sets whether the output should be formatted for readability.
	// SetPrettyPrint 设置输出是否应格式化以提高可读性。
	SetPrettyPrint(pretty bool)

	// GetPrettyPrint returns the current pretty print setting.
	// GetPrettyPrint 返回当前的格式化输出设置。
	GetPrettyPrint() bool

	// BuildCustomRPCResponse builds a response for a custom RPC method.
	// BuildCustomRPCResponse 为自定义 RPC 方法构建响应。
	BuildCustomRPCResponse(ctx context.Context, method string, result interface{}) ([]byte, error)
}
