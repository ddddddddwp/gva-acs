package handler

import (
	"net/http"
	"testing"
)

func TestClientIPResolverDirectAndUntrustedForwarding(t *testing.T) {
	resolver, err := NewClientIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	direct, _ := http.NewRequest(http.MethodPost, "http://example.com/acs", nil)
	direct.RemoteAddr = "[2001:db8::10]:7547"
	if got := resolver.Resolve(direct); got != "2001:db8::10" {
		t.Fatalf("direct IPv6=%q", got)
	}

	untrusted, _ := http.NewRequest(http.MethodPost, "http://example.com/acs", nil)
	untrusted.RemoteAddr = "192.0.2.5:1234"
	untrusted.Header.Set("X-Forwarded-For", "203.0.113.7")
	untrusted.Header.Set("X-Real-IP", "198.51.100.8")
	if got := resolver.Resolve(untrusted); got != "192.0.2.5" {
		t.Fatalf("untrusted proxy spoof accepted: %q", got)
	}
}

func TestClientIPResolverTrustedProxyAndInvalidForwardedValue(t *testing.T) {
	resolver, err := NewClientIPResolver([]string{"10.0.0.0/8", "2001:db8:ffff::/48"})
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	trusted, _ := http.NewRequest(http.MethodPost, "http://example.com/acs", nil)
	trusted.RemoteAddr = "10.0.0.9:1234"
	trusted.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.8")
	if got := resolver.Resolve(trusted); got != "203.0.113.1" {
		t.Fatalf("trusted forwarded IP=%q", got)
	}

	invalid, _ := http.NewRequest(http.MethodPost, "http://example.com/acs", nil)
	invalid.RemoteAddr = "10.0.0.9:1234"
	invalid.Header.Set("X-Forwarded-For", "not-an-ip")
	invalid.Header.Set("X-Real-IP", "also-invalid")
	if got := resolver.Resolve(invalid); got != "10.0.0.9" {
		t.Fatalf("invalid forwarded value did not fall back to peer: %q", got)
	}
}

func TestNewClientIPResolverRejectsInvalidCIDR(t *testing.T) {
	if _, err := NewClientIPResolver([]string{"not-a-cidr"}); err == nil {
		t.Fatal("invalid trusted proxy CIDR accepted")
	}
}
