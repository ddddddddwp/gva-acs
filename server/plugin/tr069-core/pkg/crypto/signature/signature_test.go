// Package signature provides digital signature verification for TR069 messages.
// 包 signature 为 TR069 消息提供数字签名验证功能。
package signature

import (
"crypto/rand"
"crypto/rsa"
"testing"
)

// TestVerifySignature 测试签名验证功能
func TestVerifySignature(t *testing.T) {
// 生成测试用的RSA密钥对
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
t.Fatalf("Failed to generate RSA key: %v", err)
}
publicKey := &privateKey.PublicKey

// 测试数据
testData := []byte("This is a test message for TR069 signature verification")

// 生成签名
signature, err := SignMessage(testData, privateKey)
if err != nil {
t.Fatalf("Failed to sign message: %v", err)
}

// 验证签名
err = VerifySignature(testData, signature, publicKey)
if err != nil {
t.Errorf("Failed to verify signature: %v", err)
}

// 测试篡改数据的验证失败情况
tamperedData := []byte("This is a tampered message")
err = VerifySignature(tamperedData, signature, publicKey)
if err == nil {
t.Error("Expected verification failure for tampered data, but got success")
}
}

// TestSignMessage 测试消息签名功能
func TestSignMessage(t *testing.T) {
// 生成测试用的RSA密钥对
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
t.Fatalf("Failed to generate RSA key: %v", err)
}

// 测试数据
testData := []byte("This is a test message for TR069 signing")

// 生成签名
signature, err := SignMessage(testData, privateKey)
if err != nil {
t.Fatalf("Failed to sign message: %v", err)
}

// 验证签名不为空
if len(signature) == 0 {
t.Error("Expected non-empty signature, but got empty")
}

// 使用公钥验证签名
err = VerifySignature(testData, signature, &privateKey.PublicKey)
if err != nil {
t.Errorf("Failed to verify generated signature: %v", err)
}
}
