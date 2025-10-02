// Package tls provides TLS configuration management for the TR069 library.
package tls

import (
 "crypto/tls"
 "testing"
 
 "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

func TestLoadTLSConfig(t *testing.T) {
 // Test with nil setup
 _, err := LoadTLSConfig(nil)
 if err == nil {
  t.Error("Expected error when setup is nil, got nil")
 }
 
 // Test with empty setup
 setup := &interfaces.TLSSetup{}
 config, err := LoadTLSConfig(setup)
 if err != nil {
  t.Errorf("Unexpected error: %v", err)
 }
 
 if config == nil {
  t.Error("Expected config, got nil")
 }
 
 // Test with server name
 setup.ServerName = "example.com"
 config, err = LoadTLSConfig(setup)
 if err != nil {
  t.Errorf("Unexpected error: %v", err)
 }
 
 if config.ServerName != "example.com" {
  t.Errorf("Expected server name 'example.com', got '%s'", config.ServerName)
 }
}

func TestValidateTLSConfig(t *testing.T) {
 // Test with nil config
 err := ValidateTLSConfig(nil)
 if err == nil {
  t.Error("Expected error when config is nil, got nil")
 }
 
 // Test with empty config
 config := &tls.Config{}
 err = ValidateTLSConfig(config)
 if err != nil {
  t.Errorf("Unexpected error for empty config: %v", err)
 }
}