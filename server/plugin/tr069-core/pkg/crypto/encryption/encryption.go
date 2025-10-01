// Package encryption provides parameter encryption and decryption for TR069 messages.
// 包 encryption 为 TR069 消息提供参数加解密功能。
package encryption

import (
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"fmt"
"io"
)

// EncryptParameter encrypts a parameter value using AES encryption.
// 使用 AES 加密参数值。
func EncryptParameter(plaintext []byte, key []byte) ([]byte, error) {
// Create a new AES cipher
block, err := aes.NewCipher(key)
if err != nil {
return nil, fmt.Errorf("failed to create AES cipher: %w", err)
}

// Generate a random IV
iv := make([]byte, aes.BlockSize)
if _, err := io.ReadFull(rand.Reader, iv); err != nil {
return nil, fmt.Errorf("failed to generate IV: %w", err)
}

// Pad the plaintext to be a multiple of the block size
paddedPlaintext := pad(plaintext, aes.BlockSize)

// Create the cipher text buffer
ciphertext := make([]byte, aes.BlockSize+len(paddedPlaintext))

// Copy the IV to the beginning of the ciphertext
copy(ciphertext[:aes.BlockSize], iv)

// Create a CBC mode encrypter
mode := cipher.NewCBCEncrypter(block, iv)

// Encrypt the padded plaintext
mode.CryptBlocks(ciphertext[aes.BlockSize:], paddedPlaintext)

return ciphertext, nil
}

// DecryptParameter decrypts a parameter value using AES decryption.
// 使用 AES 解密参数值。
func DecryptParameter(ciphertext []byte, key []byte) ([]byte, error) {
// Create a new AES cipher
block, err := aes.NewCipher(key)
if err != nil {
return nil, fmt.Errorf("failed to create AES cipher: %w", err)
}

// Check if the ciphertext is long enough
if len(ciphertext) < aes.BlockSize {
return nil, fmt.Errorf("ciphertext too short")
}

// Extract the IV from the beginning of the ciphertext
iv := ciphertext[:aes.BlockSize]
ciphertext = ciphertext[aes.BlockSize:]

// Create a CBC mode decrypter
mode := cipher.NewCBCDecrypter(block, iv)

// Decrypt the ciphertext
mode.CryptBlocks(ciphertext, ciphertext)

// Unpad the decrypted plaintext
plaintext, err := unpad(ciphertext)
if err != nil {
return nil, fmt.Errorf("failed to unpad decrypted data: %w", err)
}

return plaintext, nil
}

// pad pads the data to be a multiple of the block size using PKCS#7 padding.
// 使用 PKCS#7 填充将数据填充到块大小的倍数。
func pad(data []byte, blockSize int) []byte {
padding := blockSize - len(data)%blockSize
padtext := make([]byte, padding)
for i := range padtext {
padtext[i] = byte(padding)
}
return append(data, padtext...)
}

// unpad removes the PKCS#7 padding from the data.
// 从数据中移除 PKCS#7 填充。
func unpad(data []byte) ([]byte, error) {
length := len(data)
if length == 0 {
return nil, fmt.Errorf("invalid padding size")
}

padding := int(data[length-1])
if padding > length {
return nil, fmt.Errorf("invalid padding size")
}

return data[:length-padding], nil
}
