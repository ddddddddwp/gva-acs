// Package encryption provides parameter encryption and decryption for TR069 messages.
// 包 encryption 为 TR069 消息提供参数加解密功能。
package encryption

import (
"bytes"
"crypto/rand"
"testing"
)

// TestEncryptDecryptParameter 测试参数加密和解密功能
func TestEncryptDecryptParameter(t *testing.T) {
// 生成一个 AES 密钥 (128 bits = 16 bytes)
key := make([]byte, 16)
_, err := rand.Read(key)
if err != nil {
t.Fatalf("Failed to generate random key: %v", err)
}

// 测试数据
testData := []byte("This is a sensitive parameter value for TR069")

// 加密数据
ciphertext, err := EncryptParameter(testData, key)
if err != nil {
t.Fatalf("Failed to encrypt parameter: %v", err)
}

// 验证密文不为空且长度合理
if len(ciphertext) == 0 {
t.Error("Expected non-empty ciphertext, but got empty")
}

// 解密数据
decryptedData, err := DecryptParameter(ciphertext, key)
if err != nil {
t.Fatalf("Failed to decrypt parameter: %v", err)
}

// 验证解密后的数据与原始数据一致
if !bytes.Equal(testData, decryptedData) {
t.Errorf("Decrypted data does not match original. Expected: %s, Got: %s", 
string(testData), string(decryptedData))
}
}

// TestDecryptWithWrongKey 测试使用错误密钥解密的情况
func TestDecryptWithWrongKey(t *testing.T) {
	// 生成一个 AES 密钥 (128 bits = 16 bytes)
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}
	
	// 生成一个错误的密钥
	wrongKey := make([]byte, 16)
	_, err = rand.Read(wrongKey)
	if err != nil {
		t.Fatalf("Failed to generate wrong key: %v", err)
	}
	
	// 测试数据
	testData := []byte("This is a sensitive parameter value for TR069")
	
	// 加密数据
	ciphertext, err := EncryptParameter(testData, key)
	if err != nil {
		t.Fatalf("Failed to encrypt parameter: %v", err)
	}
	
	// 尝试使用错误的密钥解密
	_, err = DecryptParameter(ciphertext, wrongKey)
	if err == nil {
		// 解密可能不会失败，因为CBC模式下错误的密钥可能仍然产生"有效的"填充
		// 但我们可以通过检查解密后的数据是否与原始数据不同来验证
		decryptedWithWrongKey, _ := DecryptParameter(ciphertext, wrongKey)
		if bytes.Equal(testData, decryptedWithWrongKey) {
			t.Error("Expected decrypted data to be different when using wrong key")
		}
	}
}

// TestPadUnpad 测试填充和去填充功能
func TestPadUnpad(t *testing.T) {
// 测试数据
testData := []byte("Test data")

// 填充数据
paddedData := pad(testData, 16)

// 验证填充后的长度是块大小的倍数
if len(paddedData)%16 != 0 {
t.Errorf("Padded data length is not a multiple of block size. Length: %d", len(paddedData))
}

// 去填充数据
unpaddedData, err := unpad(paddedData)
if err != nil {
t.Fatalf("Failed to unpad data: %v", err)
}

// 验证去填充后的数据与原始数据一致
if !bytes.Equal(testData, unpaddedData) {
t.Errorf("Unpadded data does not match original. Expected: %s, Got: %s", 
string(testData), string(unpaddedData))
}
}
