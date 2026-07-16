package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoConnectionRequestRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second)
	if status != http.StatusUnauthorized || err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("doConnectionRequest() = (%d, %v), want 401 error", status, err)
	}
}
