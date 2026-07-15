package utils

import (
	"testing"
	"time"
)

func TestTriggerConfigChangeUsesUnlockedHandlerSnapshot(t *testing.T) {
	events := &SystemEvents{}
	firstCalls := 0
	secondCalls := 0
	events.RegisterConfigChangeHandler(func() {
		firstCalls++
		events.RegisterConfigChangeHandler(func() {
			secondCalls++
		})
	})

	done := make(chan struct{})
	go func() {
		events.TriggerConfigChange()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("TriggerConfigChange held its lock while invoking a handler")
	}

	if firstCalls != 1 || secondCalls != 0 {
		t.Fatalf("first trigger calls=(%d,%d), want=(1,0)", firstCalls, secondCalls)
	}

	events.TriggerConfigChange()
	if firstCalls != 2 || secondCalls != 1 {
		t.Fatalf("second trigger calls=(%d,%d), want=(2,1)", firstCalls, secondCalls)
	}
}
