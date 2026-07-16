package core

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/utils"
)

type shutdownTestServer struct{ events *[]string }

func (*shutdownTestServer) ListenAndServe() error { return nil }
func (s *shutdownTestServer) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		panic("shutdown context has no deadline")
	}
	*s.events = append(*s.events, "http")
	return nil
}

func TestShutdownServerDrainsHTTPBeforePluginCleanup(t *testing.T) {
	order := []string{}
	events := &utils.SystemEvents{}
	events.RegisterShutdownHandler(func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("cleanup context has no deadline")
		}
		order = append(order, "plugins")
		return nil
	})
	if err := shutdownServer(&shutdownTestServer{events: &order}, events, time.Second); err != nil {
		t.Fatalf("shutdownServer() error: %v", err)
	}
	if want := []string{"http", "plugins"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("shutdown order = %#v, want %#v", order, want)
	}
}
