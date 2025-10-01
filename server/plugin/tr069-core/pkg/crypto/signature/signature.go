// Package signature provides digital signature verification for TR069 messages.
// 包 signature 为 TR069 消息提供数字签名验证功能。
package signature

import (
"crypto"
"crypto/rand"
"crypto/rsa"
"crypto/sha256"
"fmt"
)

// VerifySignature verifies the digital signature of a TR069 message.
// 验证 TR069 消息的数字签名。
func VerifySignature(data, signature []byte, publicKey *rsa.PublicKey) error {
// Hash the data using SHA-256
hashed := sha256.Sum256(data)

// Verify the signature using RSA and PKCS#1 v1.5
err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
if err != nil {
return fmt.Errorf("signature verification failed: %w", err)
}

return nil
}

// SignMessage signs a TR069 message using the provided private key.
// 使用提供的私钥对 TR069 消息进行签名。
func SignMessage(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
// Hash the data using SHA-256
hashed := sha256.Sum256(data)

// Sign the hash using RSA and PKCS#1 v1.5
signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
if err != nil {
return nil, fmt.Errorf("failed to sign message: %w", err)
}

return signature, nil
}
