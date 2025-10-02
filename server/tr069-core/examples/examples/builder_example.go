package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// BuildTR069Message 演示如何使用构建器构建TR069消息
func BuildTR069Message() {
	// 创建构建器实例
	builder := factory.NewBuilder()

	// 创建一个InformResponse消息
	msg := &interfaces.Message{
		Method: "InformResponse",
		Parameters: []interfaces.Parameter{
			{
				Name:  "MaxEnvelopes",
				Value: 1,
				Type:  "int",
			},
		},
	}

	// 构建消息
	data, err := builder.BuildMessage(context.Background(), msg)
	if err != nil {
		log.Fatalf("构建消息失败: %v", err)
	}

	// 输出构建结果
	fmt.Printf("构建的TR069消息:\n%s\n", string(data))
}

// BuildTR069Fault 演示如何构建TR069错误响应
func BuildTR069Fault() {
	// 创建构建器实例
	builder := factory.NewBuilder()

	// 构建错误响应（例如：9001 - Request denied）
	faultResponse, err := builder.BuildFault(context.Background(), 9001, "Request denied")
	if err != nil {
		log.Fatalf("构建错误响应失败: %v", err)
	}

	// 输出构建结果
	fmt.Printf("构建的TR069错误响应:\n%s\n", string(faultResponse))
}

// 示例函数调用
func ExampleBuildTR069Message() {
	BuildTR069Message()
	BuildTR069Fault()
}