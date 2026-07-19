package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
)

func TestCWMPErrorStatusMapsDeletingDeviceToConflict(t *testing.T) {
	if got := cwmpErrorStatus(fmt.Errorf("wrapped: %w", service.ErrDeviceDeleting)); got != http.StatusConflict {
		t.Fatalf("cwmpErrorStatus() = %d, want %d", got, http.StatusConflict)
	}
	if got := cwmpErrorStatus(errors.New("other")); got != http.StatusInternalServerError {
		t.Fatalf("cwmpErrorStatus(other) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestRemoteIPFromRequest(t *testing.T) {
	r1, _ := http.NewRequest(http.MethodPost, "http://example.com/", nil)
	r1.RemoteAddr = "10.0.0.9:1234"
	r1.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.9")
	if ip := remoteIPFromRequest(r1); ip != "10.0.0.9" {
		t.Fatalf("expected untrusted xff to be ignored, got %q", ip)
	}

	r2, _ := http.NewRequest(http.MethodPost, "http://example.com/", nil)
	r2.RemoteAddr = "10.0.0.9:1234"
	r2.Header.Set("X-Real-Ip", "198.51.100.2")
	if ip := remoteIPFromRequest(r2); ip != "10.0.0.9" {
		t.Fatalf("expected untrusted x-real-ip to be ignored, got %q", ip)
	}

	r3, _ := http.NewRequest(http.MethodPost, "http://example.com/", nil)
	r3.RemoteAddr = "10.0.0.9:1234"
	if ip := remoteIPFromRequest(r3); ip != "10.0.0.9" {
		t.Fatalf("expected remoteaddr host, got %q", ip)
	}
}
