// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

// Config is the interface for TR069 library configuration.
// Config 是 TR069 库配置的接口。
type Config interface {
 // GetStrictMode returns whether the parser should operate in strict mode.
 // GetStrictMode 返回解析器是否应在严格模式下运行。
 GetStrictMode() bool
 
 // SetStrictMode sets whether the parser should operate in strict mode.
 // SetStrictMode 设置解析器是否应在严格模式下运行。
 SetStrictMode(strict bool)
 
 // GetPrettyPrint returns whether the builder should format output for readability.
 // GetPrettyPrint 返回构建器是否应格式化输出以提高可读性。
 GetPrettyPrint() bool
 
 // SetPrettyPrint sets whether the builder should format output for readability.
 // SetPrettyPrint 设置构建器是否应格式化输出以提高可读性。
 SetPrettyPrint(pretty bool)
 
 // GetMaxDepth returns the maximum parsing depth.
 // GetMaxDepth 返回最大解析深度。
 GetMaxDepth() int
 
 // SetMaxDepth sets the maximum parsing depth.
 // SetMaxDepth 设置最大解析深度。
 SetMaxDepth(depth int)
 
 // GetValidation returns whether validation is enabled.
 // GetValidation 返回是否启用了验证。
 GetValidation() bool
 
 // SetValidation sets whether validation is enabled.
 // SetValidation 设置是否启用验证。
 SetValidation(validate bool)
 
 // GetSecurityConfig returns the security configuration.
 // GetSecurityConfig 返回安全配置。
 GetSecurityConfig() *SecurityConfig
 
 // SetSecurityConfig sets the security configuration.
 // SetSecurityConfig 设置安全配置。
 SetSecurityConfig(securityConfig *SecurityConfig)
}

// SecurityConfig 安全配置结构体
type SecurityConfig struct {
    // EnableTLS 是否启用TLS
    EnableTLS bool
    // TLSSetup TLS配置
    TLSSetup *TLSSetup
    // EnableSignatureVerification 是否启用签名验证
    EnableSignatureVerification bool
    // EnableParameterEncryption 是否启用参数加密
    EnableParameterEncryption bool
    // EncryptionKey 加密密钥
    EncryptionKey []byte
}

// TLSSetup TLS配置结构体
type TLSSetup struct {
    // CertFile 证书文件路径
    CertFile string
    // KeyFile 私钥文件路径
    KeyFile string
    // CAFile CA证书文件路径
    CAFile string
    // ServerName 服务器名称
    ServerName string
    // InsecureSkipVerify 是否跳过证书验证
    InsecureSkipVerify bool
}

// Option is a function that configures a Config.
// Option 是用于配置 Config 的函数。
type Option func(Config)

// WithStrictMode returns an Option that sets the strict mode.
// WithStrictMode 返回一个设置严格模式的 Option。
func WithStrictMode(strict bool) Option {
 return func(c Config) {
  c.SetStrictMode(strict)
 }
}

// WithPrettyPrint returns an Option that sets the pretty print mode.
// WithPrettyPrint 返回一个设置格式化输出模式的 Option。
func WithPrettyPrint(pretty bool) Option {
 return func(c Config) {
  c.SetPrettyPrint(pretty)
 }
}

// WithMaxDepth returns an Option that sets the maximum parsing depth.
// WithMaxDepth 返回一个设置最大解析深度的 Option。
func WithMaxDepth(depth int) Option {
 return func(c Config) {
  c.SetMaxDepth(depth)
 }
}

// WithValidation returns an Option that sets the validation mode.
// WithValidation 返回一个设置验证模式的 Option。
func WithValidation(validate bool) Option {
 return func(c Config) {
  c.SetValidation(validate)
 }
}

// WithSecurityConfig 配置安全选项
func WithSecurityConfig(securityConfig *SecurityConfig) Option {
    return func(c Config) {
        c.SetSecurityConfig(securityConfig)
    }
}