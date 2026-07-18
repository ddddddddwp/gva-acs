package trace

import (
	"context"
	"testing"
	"time"
)

func TestTraceIDContextAndStoreUseTraceSemantics(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-1")
	if got := TraceID(ctx); got != "trace-1" {
		t.Fatalf("TraceID() = %q, want trace-1", got)
	}

	previous := Default
	Default = NewStore(10, time.Minute)
	t.Cleanup(func() { Default = previous })
	Add(ctx, "parse", "parsed", map[string]string{"method": "Inform"})
	entries := Get(ctx, "trace-1")
	if len(entries) != 1 || entries[0].Stage != "parse" {
		t.Fatalf("trace entries = %#v", entries)
	}
}
