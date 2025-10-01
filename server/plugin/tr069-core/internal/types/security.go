// Package types defines internal data types for the TR069 library.
package types

// SecurityConfig holds security-related configuration.
// SecurityConfig 保存安全相关的配置。
type SecurityConfig struct {
	// EnableSignatureVerification enables message signature verification.
	// EnableSignatureVerification 启用消息签名验证。
	EnableSignatureVerification bool
	
	// SignatureKey is the key used for signature verification.
	// SignatureKey 是用于签名验证的密钥。
	SignatureKey []byte
	
	// EnableParameterEncryption enables parameter encryption.
	// EnableParameterEncryption 启用参数加密。
	EnableParameterEncryption bool
	
	// EncryptionKey is the key used for parameter encryption.
	// EncryptionKey 是用于参数加密的密钥。
	EncryptionKey []byte
	
	// SensitiveParameters is a list of parameter names that should be encrypted.
	// SensitiveParameters 是应加密的参数名称列表。
	SensitiveParameters []string
}

// IsSensitiveParameter checks if a parameter should be encrypted.
// IsSensitiveParameter 检查参数是否应加密。
func (s *SecurityConfig) IsSensitiveParameter(name string) bool {
	for _, param := range s.SensitiveParameters {
		if param == name {
			return true
		}
	}
	return false
}