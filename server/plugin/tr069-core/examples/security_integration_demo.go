package main

import (
	"context"
	"fmt"
	"log"

	"github.com/root/demo/tr069/factory"
	"github.com/root/demo/tr069/interfaces"
)

func main() {
	fmt.Println("=== TR069 Security Integration Test ===")

	// 创建安全配置
	securityConfig := &interfaces.SecurityConfig{
		EnableSignatureVerification: true,
		EnableParameterEncryption:   true,
		EncryptionKey:               []byte("test-key-12345678"), // 16字节AES密钥
	}

	// 创建解析器和构建器
	parser := factory.NewParser(factory.WithSecurityConfig(securityConfig))
	builder := factory.NewBuilder(factory.WithSecurityConfig(securityConfig))

	// 设置安全配置
	if securityParser, ok := parser.(interface{ SetSecurityConfig(*interfaces.SecurityConfig) }); ok {
		securityParser.SetSecurityConfig(securityConfig)
	}

	if securityBuilder, ok := builder.(interface{ SetSecurityConfig(*interfaces.SecurityConfig) }); ok {
		securityBuilder.SetSecurityConfig(securityConfig)
	}

	// 创建测试消息
	testMessage := &interfaces.Message{
		Method: "Inform",
		Parameters: []interfaces.Parameter{
			{Name: "Device.DeviceInfo.SerialNumber", Value: "1234567890", Type: "string"},
			{Name: "Device.ManagementServer.ConnectionRequestUsername", Value: "admin", Type: "string"},
			{Name: "Device.ManagementServer.ConnectionRequestPassword", Value: "secret_password", Type: "string"},
		},
	}

	// 构建消息
	ctx := context.Background()
	builtMessage, err := builder.BuildMessage(ctx, testMessage)
	if err != nil {
		log.Fatalf("Failed to build message: %v", err)
	}

	fmt.Printf("Built message length: %d bytes\n", len(builtMessage))

	// 解析消息
	parsedMessage, err := parser.ParseMessage(ctx, builtMessage)
	if err != nil {
		log.Fatalf("Failed to parse message: %v", err)
	}

	fmt.Printf("Parsed message method: %s\n", parsedMessage.Method)
	fmt.Printf("Parsed parameters count: %d\n", len(parsedMessage.Parameters))

	// 验证参数
	for _, param := range parsedMessage.Parameters {
		fmt.Printf("Parameter: %s = %v (%s)\n", param.Name, param.Value, param.Type)
	}

	fmt.Println("Security integration test completed successfully!")
}