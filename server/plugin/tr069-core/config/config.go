// Package config provides configuration management for the TR069 library.
package config

import (
 "sync"
 
 "github.com/root/demo/tr069/interfaces"
)

// Config implements the interfaces.Config interface.
type Config struct {
 mu           sync.RWMutex
 strictMode   bool
 prettyPrint  bool
 maxDepth     int
 validation   bool
 securityConfig *interfaces.SecurityConfig
}

// New creates a new Config with default values.
func New() *Config {
 return &Config{
  strictMode:  false,
  prettyPrint: false,
  maxDepth:    100,
  validation:  true,
  securityConfig: &interfaces.SecurityConfig{
   EnableTLS: false,
   EnableSignatureVerification: false,
   EnableParameterEncryption: false,
  },
 }
}

// GetStrictMode returns whether the parser should operate in strict mode.
func (c *Config) GetStrictMode() bool {
 c.mu.RLock()
 defer c.mu.RUnlock()
 return c.strictMode
}

// SetStrictMode sets whether the parser should operate in strict mode.
func (c *Config) SetStrictMode(strict bool) {
 c.mu.Lock()
 defer c.mu.Unlock()
 c.strictMode = strict
}

// GetPrettyPrint returns whether the builder should format output for readability.
func (c *Config) GetPrettyPrint() bool {
 c.mu.RLock()
 defer c.mu.RUnlock()
 return c.prettyPrint
}

// SetPrettyPrint sets whether the builder should format output for readability.
func (c *Config) SetPrettyPrint(pretty bool) {
 c.mu.Lock()
 defer c.mu.Unlock()
 c.prettyPrint = pretty
}

// GetMaxDepth returns the maximum parsing depth.
func (c *Config) GetMaxDepth() int {
 c.mu.RLock()
 defer c.mu.RUnlock()
 return c.maxDepth
}

// SetMaxDepth sets the maximum parsing depth.
func (c *Config) SetMaxDepth(depth int) {
 c.mu.Lock()
 defer c.mu.Unlock()
 c.maxDepth = depth
}

// GetValidation returns whether validation is enabled.
func (c *Config) GetValidation() bool {
 c.mu.RLock()
 defer c.mu.RUnlock()
 return c.validation
}

// SetValidation sets whether validation is enabled.
func (c *Config) SetValidation(validate bool) {
 c.mu.Lock()
 defer c.mu.Unlock()
 c.validation = validate
}

// GetSecurityConfig returns the security configuration.
func (c *Config) GetSecurityConfig() *interfaces.SecurityConfig {
 c.mu.RLock()
 defer c.mu.RUnlock()
 return c.securityConfig
}

// SetSecurityConfig sets the security configuration.
func (c *Config) SetSecurityConfig(securityConfig *interfaces.SecurityConfig) {
 c.mu.Lock()
 defer c.mu.Unlock()
 c.securityConfig = securityConfig
}

// ApplyOptions applies the given options to the config.
func (c *Config) ApplyOptions(opts ...interfaces.Option) {
 for _, opt := range opts {
  opt(c)
 }
}

// WithSecurityConfig 配置安全选项
func WithSecurityConfig(securityConfig *interfaces.SecurityConfig) interfaces.Option {
    return func(c interfaces.Config) {
        c.SetSecurityConfig(securityConfig)
    }
}