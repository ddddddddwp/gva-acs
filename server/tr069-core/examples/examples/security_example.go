package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/tls"
)

// SecurityConfigExample 演示如何配置和使用TR069安全功能
func SecurityConfigExample() {
	// 创建安全配置
	securityConfig := &interfaces.SecurityConfig{
		EnableTLS: true,
		TLSSetup: &interfaces.TLSSetup{
			ServerName: "example.com",
			// 在实际使用中，需要提供有效的证书和密钥文件路径
			// CertFile: "path/to/cert.pem",
			// KeyFile:  "path/to/key.pem",
			// CAFile:   "path/to/ca.pem",
			InsecureSkipVerify: true, // 仅用于测试环境
		},
		EnableSignatureVerification: false, // 签名验证将在后续实现
		EnableParameterEncryption:   false, // 参数加密将在后续实现
	}

	// 使用安全配置创建解析器
	parser := factory.NewParser(
		factory.WithSecurityConfig(securityConfig),
	)

	// 输出配置信息
	config, ok := parser.(interfaces.Config)
	if ok {
		secConfig := config.GetSecurityConfig()
		fmt.Printf("TLS启用状态: %v\n", secConfig.EnableTLS)
		fmt.Printf("签名验证启用状态: %v\n", secConfig.EnableSignatureVerification)
		fmt.Printf("参数加密启用状态: %v\n", secConfig.EnableParameterEncryption)

		if secConfig.TLSSetup != nil {
			fmt.Printf("服务器名称: %s\n", secConfig.TLSSetup.ServerName)
			fmt.Printf("跳过证书验证: %v\n", secConfig.TLSSetup.InsecureSkipVerify)
		}
	}

	// 示例TR069 XML消息
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
	fmt.Printf("RPC方法: %s\n", message.Method)
	fmt.Printf("参数数量: %d\n", len(message.Parameters))

	// 输出参数详情
	for _, param := range message.Parameters {
		fmt.Printf("参数名: %s, 值: %v, 类型: %s\n", param.Name, param.Value, param.Type)
	}
}

// TLSConfigExample 演示如何使用TLS配置管理模块
func TLSConfigExample() {
	// 创建TLS设置
	tlsSetup := &interfaces.TLSSetup{
		ServerName:         "example.com",
		InsecureSkipVerify: true, // 仅用于测试环境
		// 在实际使用中，需要提供有效的证书和密钥文件路径
		// CertFile: "path/to/cert.pem",
		// KeyFile:  "path/to/key.pem",
		// CAFile:   "path/to/ca.pem",
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

	fmt.Printf("TLS配置加载成功\n")
	fmt.Printf("服务器名称: %s\n", tlsConfig.ServerName)
	fmt.Printf("跳过证书验证: %v\n", tlsConfig.InsecureSkipVerify)
}

// SecurityExample 演示如何配置和使用TR069安全功能
func SecurityExample() {
	fmt.Println("=== 安全配置示例 ===")
	SecurityConfigExample()

	fmt.Println("\n=== TLS配置示例 ===")
	TLSConfigExample()
}
