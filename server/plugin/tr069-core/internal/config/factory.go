// Package config provides configuration factory methods.
package config

import (
	"github.com/root/demo/tr069/interfaces"
)

// CreateDefaultConfig creates a default configuration.
func CreateDefaultConfig() interfaces.Config {
	return &defaultConfig{
		strictMode:     false,
		prettyPrint:    false,
		maxDepth:       100,
		validation:     true,
		securityConfig: &interfaces.SecurityConfig{
			EnableTLS:                     false,
			EnableSignatureVerification:   false,
			EnableParameterEncryption:     false,
		},
	}
}

// defaultConfig implements the Config interface.
type defaultConfig struct {
	strictMode     bool
	prettyPrint    bool
	maxDepth       int
	validation     bool
	securityConfig *interfaces.SecurityConfig
}

func (c *defaultConfig) GetStrictMode() bool {
	return c.strictMode
}

func (c *defaultConfig) SetStrictMode(strict bool) {
	c.strictMode = strict
}

func (c *defaultConfig) GetPrettyPrint() bool {
	return c.prettyPrint
}

func (c *defaultConfig) SetPrettyPrint(pretty bool) {
	c.prettyPrint = pretty
}

func (c *defaultConfig) GetMaxDepth() int {
	return c.maxDepth
}

func (c *defaultConfig) SetMaxDepth(depth int) {
	c.maxDepth = depth
}

func (c *defaultConfig) GetValidation() bool {
	return c.validation
}

func (c *defaultConfig) SetValidation(validate bool) {
	c.validation = validate
}

func (c *defaultConfig) GetSecurityConfig() *interfaces.SecurityConfig {
	return c.securityConfig
}

func (c *defaultConfig) SetSecurityConfig(securityConfig *interfaces.SecurityConfig) {
	c.securityConfig = securityConfig
}