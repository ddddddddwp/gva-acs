// Package tls provides TLS configuration management for the TR069 library.
package tls

import (
 "crypto/tls"
 "crypto/x509"
 "fmt"
 "os"
 
 "github.com/root/demo/tr069/interfaces"
)

// LoadTLSConfig loads TLS configuration from the provided setup.
func LoadTLSConfig(setup *interfaces.TLSSetup) (*tls.Config, error) {
 if setup == nil {
  return nil, fmt.Errorf("TLS setup is nil")
 }
 
 config := &tls.Config{
  ServerName:         setup.ServerName,
  InsecureSkipVerify: setup.InsecureSkipVerify,
 }
 
 // Load certificate and key if provided
 if setup.CertFile != "" && setup.KeyFile != "" {
  cert, err := tls.LoadX509KeyPair(setup.CertFile, setup.KeyFile)
  if err != nil {
   return nil, fmt.Errorf("failed to load certificate and key: %w", err)
  }
  config.Certificates = []tls.Certificate{cert}
 }
 
 // Load CA certificate if provided
 if setup.CAFile != "" {
  caCert, err := os.ReadFile(setup.CAFile)
  if err != nil {
   return nil, fmt.Errorf("failed to read CA certificate: %w", err)
  }
  
  caCertPool := x509.NewCertPool()
  if !caCertPool.AppendCertsFromPEM(caCert) {
   return nil, fmt.Errorf("failed to parse CA certificate")
  }
  config.RootCAs = caCertPool
  
  // For mutual TLS authentication
  if len(config.Certificates) > 0 {
   config.ClientAuth = tls.RequireAndVerifyClientCert
   config.ClientCAs = caCertPool
  }
 }
 
 return config, nil
}

// ValidateTLSConfig validates the TLS configuration.
func ValidateTLSConfig(config *tls.Config) error {
 if config == nil {
  return fmt.Errorf("TLS config is nil")
 }
 
 // Validate certificates if present
 for i, cert := range config.Certificates {
  if cert.Certificate == nil || len(cert.Certificate) == 0 {
   return fmt.Errorf("certificate %d is invalid", i)
  }
  
  if cert.PrivateKey == nil {
   return fmt.Errorf("private key for certificate %d is nil", i)
  }
 }
 
 return nil
}