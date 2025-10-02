// Package factory provides factory functions for creating TR069 components.
package factory

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/config"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/builder"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/parser"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/session"
)

// NewParser creates a new Parser with the given options.
func NewParser(opts ...interfaces.Option) interfaces.Parser {
	cfg := config.New()
	cfg.ApplyOptions(opts...)

	p := parser.New()
	p.SetStrictMode(cfg.GetStrictMode())

	return p
}

// NewBuilder creates a new Builder with the given options.
func NewBuilder(opts ...interfaces.Option) interfaces.Builder {
	cfg := config.New()
	cfg.ApplyOptions(opts...)

	b := builder.New()
	b.SetPrettyPrint(cfg.GetPrettyPrint())

	return b
}

// NewConfig creates a new Config with the given options.
func NewConfig(opts ...interfaces.Option) interfaces.Config {
	cfg := config.New()
	cfg.ApplyOptions(opts...)
	return cfg
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() interfaces.SessionManager {
	return session.NewSessionManager()
}

// WithSecurityConfig 配置安全选项
func WithSecurityConfig(securityConfig *interfaces.SecurityConfig) interfaces.Option {
	return func(c interfaces.Config) {
		c.SetSecurityConfig(securityConfig)
	}
}
