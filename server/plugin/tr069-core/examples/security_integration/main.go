package main

import (
"context"
"crypto/rand"
"crypto/rsa"
"fmt"
"log"

"github.com/root/demo/tr069/factory"
"github.com/root/demo/tr069/interfaces"
"github.com/root/demo/tr069/pkg/crypto/encryption"
"github.com/root/demo/tr069/pkg/crypto/signature"
)

func main() {
// 创建安全配置
securityConfig := createSecurityConfig()

// 使用工厂函数创建解析器和构建器，并配置安全选项
parser := factory.NewParser(factory.WithSecurityConfig(securityConfig))
builder := factory.NewBuilder(factory.WithSecurityConfig(securityConfig))

// 创建一个示例TR069消息
message := createSampleMessage()

// 使用构建器构建消息
ctx := context.Background()
builtMessage, err := builder.BuildMessage(ctx, message)
if err != nil {
log.Fatalf("Failed to build message: %v", err)
}

fmt.Println("Built message:")
fmt.Println(string(builtMessage))

// 使用解析器解析消息
parsedMessage, err := parser.ParseMessage(ctx, builtMessage)
if err != nil {
log.Fatalf("Failed to parse message: %v", err)
}

fmt.Printf("Parsed message method: %s\n", parsedMessage.Method)
fmt.Printf("Parsed message parameters: %v\n", parsedMessage.Parameters)

// 演示数字签名功能
demoDigitalSignature()

// 演示参数加密功能
demoParameterEncryption()
}

func createSecurityConfig() *interfaces.SecurityConfig {
// 生成一个用于加密的密钥
encryptionKey := make([]byte, 16)
_, err := rand.Read(encryptionKey)
if err != nil {
log.Fatalf("Failed to generate encryption key: %v", err)
}

return &interfaces.SecurityConfig{
EnableTLS:                   false, // 在示例中不启用TLS
EnableSignatureVerification: true,  // 启用签名验证
EnableParameterEncryption:   true,  // 启用参数加密
EncryptionKey:               encryptionKey,
}
}

func createSampleMessage() *interfaces.Message {
return &interfaces.Message{
Method: "cwmp:Inform",
Parameters: []interfaces.Parameter{
{Name: "Device.DeviceInfo.Manufacturer", Type: "string", Value: "ExampleCorp"},
{Name: "Device.DeviceInfo.ModelName", Type: "string", Value: "ModelX"},
{Name: "Device.ManagementServer.ConnectionRequestURL", Type: "string", Value: "http://example.com/crq"},
},
}
}

func demoDigitalSignature() {
fmt.Println("\n=== Digital Signature Demo ===")

// 生成RSA密钥对
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
log.Fatalf("Failed to generate RSA key: %v", err)
}
publicKey := &privateKey.PublicKey

// 要签名的数据
data := []byte("This is a TR069 message that needs to be signed")

// 签名数据
signatureData, err := signature.SignMessage(data, privateKey)
if err != nil {
log.Fatalf("Failed to sign message: %v", err)
}

fmt.Printf("Message: %s\n", string(data))
fmt.Printf("Signature length: %d bytes\n", len(signatureData))

// 验证签名
err = signature.VerifySignature(data, signatureData, publicKey)
if err != nil {
log.Fatalf("Failed to verify signature: %v", err)
}

fmt.Println("Signature verification successful!")

// 尝试验证被篡改的数据（应该失败）
tamperedData := []byte("This is a TAMPERED message")
err = signature.VerifySignature(tamperedData, signatureData, publicKey)
if err != nil {
fmt.Println("Expected signature verification failure for tampered data:", err)
} else {
log.Fatal("Signature verification should have failed for tampered data")
}
}

func demoParameterEncryption() {
fmt.Println("\n=== Parameter Encryption Demo ===")

// 生成一个AES密钥
key := make([]byte, 16)
_, err := rand.Read(key)
if err != nil {
log.Fatalf("Failed to generate random key: %v", err)
}

// 要加密的敏感参数
sensitiveData := []byte("This is a sensitive parameter value: secret_password_123")

fmt.Printf("Original data: %s\n", string(sensitiveData))

// 加密数据
encryptedData, err := encryption.EncryptParameter(sensitiveData, key)
if err != nil {
log.Fatalf("Failed to encrypt parameter: %v", err)
}

fmt.Printf("Encrypted data length: %d bytes\n", len(encryptedData))

// 解密数据
decryptedData, err := encryption.DecryptParameter(encryptedData, key)
if err != nil {
log.Fatalf("Failed to decrypt parameter: %v", err)
}

fmt.Printf("Decrypted data: %s\n", string(decryptedData))

// 验证解密后的数据与原始数据一致
if string(sensitiveData) != string(decryptedData) {
log.Fatal("Decrypted data does not match original data")
}

fmt.Println("Encryption/decryption successful!")
}
