// Package config provides configuration management for the TR069 library.
package config

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

func TestConfig(t *testing.T) {
	cfg := New()

	// Test strict mode
	cfg.SetStrictMode(true)
	if !cfg.GetStrictMode() {
		t.Error("Expected strict mode to be true")
	}

	cfg.SetStrictMode(false)
	if cfg.GetStrictMode() {
		t.Error("Expected strict mode to be false")
	}

	// Test pretty print
	cfg.SetPrettyPrint(true)
	if !cfg.GetPrettyPrint() {
		t.Error("Expected pretty print to be true")
	}

	cfg.SetPrettyPrint(false)
	if cfg.GetPrettyPrint() {
		t.Error("Expected pretty print to be false")
	}

	// Test max depth
	cfg.SetMaxDepth(50)
	if cfg.GetMaxDepth() != 50 {
		t.Errorf("Expected max depth to be 50, got %d", cfg.GetMaxDepth())
	}

	// Test validation
	cfg.SetValidation(true)
	if !cfg.GetValidation() {
		t.Error("Expected validation to be true")
	}

	cfg.SetValidation(false)
	if cfg.GetValidation() {
		t.Error("Expected validation to be false")
	}
}

func TestConfig_Options(t *testing.T) {
	cfg := New()

	// Apply options
	cfg.ApplyOptions(
		interfaces.WithStrictMode(true),
		interfaces.WithPrettyPrint(true),
		interfaces.WithMaxDepth(200),
		interfaces.WithValidation(false),
	)

	if !cfg.GetStrictMode() {
		t.Error("Expected strict mode to be true")
	}

	if !cfg.GetPrettyPrint() {
		t.Error("Expected pretty print to be true")
	}

	if cfg.GetMaxDepth() != 200 {
		t.Errorf("Expected max depth to be 200, got %d", cfg.GetMaxDepth())
	}

	if cfg.GetValidation() {
		t.Error("Expected validation to be false")
	}
}

func TestConfig_Security(t *testing.T) {
	cfg := New()

	// Test default security config
	secConfig := cfg.GetSecurityConfig()
	if secConfig == nil {
		t.Error("Expected default security config, got nil")
	}

	if secConfig.EnableTLS {
		t.Error("Expected TLS to be disabled by default")
	}

	if secConfig.EnableSignatureVerification {
		t.Error("Expected signature verification to be disabled by default")
	}

	if secConfig.EnableParameterEncryption {
		t.Error("Expected parameter encryption to be disabled by default")
	}

	// Test setting security config
	newSecConfig := &interfaces.SecurityConfig{
		EnableTLS: true,
		TLSSetup: &interfaces.TLSSetup{
			ServerName: "test.example.com",
		},
		EnableSignatureVerification: true,
		EnableParameterEncryption:   true,
	}

	cfg.SetSecurityConfig(newSecConfig)
	retrievedSecConfig := cfg.GetSecurityConfig()

	if !retrievedSecConfig.EnableTLS {
		t.Error("Expected TLS to be enabled")
	}

	if !retrievedSecConfig.EnableSignatureVerification {
		t.Error("Expected signature verification to be enabled")
	}

	if !retrievedSecConfig.EnableParameterEncryption {
		t.Error("Expected parameter encryption to be enabled")
	}

	if retrievedSecConfig.TLSSetup == nil {
		t.Error("Expected TLS setup to be set")
	} else if retrievedSecConfig.TLSSetup.ServerName != "test.example.com" {
		t.Errorf("Expected server name to be 'test.example.com', got '%s'", retrievedSecConfig.TLSSetup.ServerName)
	}
}

func TestConfig_SecurityOptions(t *testing.T) {
	cfg := New()

	// Test security config option
	secConfig := &interfaces.SecurityConfig{
		EnableTLS: true,
		TLSSetup: &interfaces.TLSSetup{
			ServerName: "option.example.com",
		},
	}

	cfg.ApplyOptions(WithSecurityConfig(secConfig))

	retrievedSecConfig := cfg.GetSecurityConfig()
	if !retrievedSecConfig.EnableTLS {
		t.Error("Expected TLS to be enabled")
	}

	if retrievedSecConfig.TLSSetup == nil {
		t.Error("Expected TLS setup to be set")
	} else if retrievedSecConfig.TLSSetup.ServerName != "option.example.com" {
		t.Errorf("Expected server name to be 'option.example.com', got '%s'", retrievedSecConfig.TLSSetup.ServerName)
	}
}
