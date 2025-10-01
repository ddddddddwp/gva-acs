package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/root/demo/tr069/factory"
	"github.com/root/demo/tr069/interfaces"
	"github.com/root/demo/tr069/pkg/tls"
)

// FullSecurityExample 演示完整的安全配置功能
func FullSecurityExample() {
	fmt.Println("=== TR069 安全配置完整示例 ===")

	// 1. 创建完整的安全配置
	securityConfig := &interfaces.SecurityConfig{
		EnableTLS: true,
		TLSSetup: &interfaces.TLSSetup{
			ServerName:         "acs.example.com",
			InsecureSkipVerify: true, // 仅用于测试环境，生产环境应设置为false
			// 在实际使用中，需要提供有效的证书和密钥文件路径
			// CertFile: "path/to/server.crt",
			// KeyFile:  "path/to/server.key",
			// CAFile:   "path/to/ca.crt",
		},
		EnableSignatureVerification: false, // 签名验证将在后续实现
		EnableParameterEncryption:   false, // 参数加密将在后续实现
	}

	// 2. 使用安全配置创建解析器和构建器
	parser := factory.NewParser(
		factory.WithSecurityConfig(securityConfig),
	)

	builder := factory.NewBuilder(
		factory.WithSecurityConfig(securityConfig),
	)

	// 3. 验证安全配置是否正确应用
	configParser, ok := parser.(interfaces.Config)
	if ok {
		secConfig := configParser.GetSecurityConfig()
		fmt.Printf("解析器安全配置:\n")
		fmt.Printf("  TLS启用: %v\n", secConfig.EnableTLS)
		fmt.Printf("  签名验证启用: %v\n", secConfig.EnableSignatureVerification)
		fmt.Printf("  参数加密启用: %v\n", secConfig.EnableParameterEncryption)
		
		if secConfig.TLSSetup != nil {
			fmt.Printf("  TLS服务器名称: %s\n", secConfig.TLSSetup.ServerName)
			fmt.Printf("  跳过证书验证: %v\n", secConfig.TLSSetup.InsecureSkipVerify)
		}
	}

	configBuilder, ok := builder.(interfaces.Config)
	if ok {
		secConfig := configBuilder.GetSecurityConfig()
		fmt.Printf("\n构建器安全配置:\n")
		fmt.Printf("  TLS启用: %v\n", secConfig.EnableTLS)
		fmt.Printf("  签名验证启用: %v\n", secConfig.EnableSignatureVerification)
		fmt.Printf("  参数加密启用: %v\n", secConfig.EnableParameterEncryption)
		
		if secConfig.TLSSetup != nil {
			fmt.Printf("  TLS服务器名称: %s\n", secConfig.TLSSetup.ServerName)
			fmt.Printf("  跳过证书验证: %v\n", secConfig.TLSSetup.InsecureSkipVerify)
		}
	}

	// 4. 演示TLS配置管理模块的使用
	fmt.Printf("\n=== TLS配置管理示例 ===\n")
	tlsSetup := &interfaces.TLSSetup{
		ServerName:         "tls.example.com",
		InsecureSkipVerify: true,
	}

	// 加载TLS配置
	tlsConfig, err := tls.LoadTLSConfig(tlsSetup)
	if err != nil {
		log.Fatalf("加载TLS配置失败: %v", err)
	}

	// 验证TLS配置
	if err := tls.ValidateTLSConfig(tlsConfig); err != nil {
		log.Fatalf("验证TLS配置失败: %v", err)
	}

	fmt.Printf("TLS配置加载成功:\n")
	fmt.Printf("  服务器名称: %s\n", tlsConfig.ServerName)
	fmt.Printf("  跳过证书验证: %v\n", tlsConfig.InsecureSkipVerify)

	// 5. 使用解析器解析示例消息
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
	fmt.Printf("\n=== 消息解析结果 ===\n")
	fmt.Printf("RPC方法: %s\n", message.Method)
	fmt.Printf("参数数量: %d\n", len(message.Parameters))

	// 输出参数详情
	for _, param := range message.Parameters {
		fmt.Printf("参数名: %s, 值: %v, 类型: %s\n", param.Name, param.Value, param.Type)
	}

	// 6. 使用构建器构建响应消息
	responseMessage := &interfaces.Message{
		Method: "InformResponse",
		Parameters: []interfaces.Parameter{
			{
				Name:  "MaxEnvelopes",
				Value: 1,
				Type:  "xsd:int",
			},
		},
	}

	responseData, err := builder.BuildMessage(context.Background(), responseMessage)
	if err != nil {
		log.Fatalf("构建响应消息失败: %v", err)
	}

	fmt.Printf("\n=== 构建的响应消息 ===\n")
	fmt.Printf("%s\n", string(responseData))
}