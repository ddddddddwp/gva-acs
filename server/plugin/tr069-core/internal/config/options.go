// Package config provides configuration options.
package config

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// Option represents a configuration option.
type Option func(interfaces.Config)

// WithStrictMode sets the strict mode option.
func WithStrictMode(strict bool) Option {
	return func(c interfaces.Config) {
		c.SetStrictMode(strict)
	}
}

// WithPrettyPrint sets the pretty print option.
func WithPrettyPrint(pretty bool) Option {
	return func(c interfaces.Config) {
		c.SetPrettyPrint(pretty)
	}
}

// WithMaxDepth sets the maximum depth option.
func WithMaxDepth(depth int) Option {
	return func(c interfaces.Config) {
		c.SetMaxDepth(depth)
	}
}

// WithValidation sets the validation option.
func WithValidation(enabled bool) Option {
	return func(c interfaces.Config) {
		c.SetValidation(enabled)
	}
}

// ApplyOptions applies the given options to the configuration.
func ApplyOptions(config interfaces.Config, options ...Option) {
	for _, option := range options {
		option(config)
	}
}