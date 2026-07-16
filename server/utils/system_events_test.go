package utils

import (
	"context"
	"errors"
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

func TestTriggerShutdownUsesUnlockedHandlerSnapshot(t *testing.T) {
	events := &SystemEvents{}
	firstCalls := 0
	secondCalls := 0
	wantErr := errors.New("shutdown failed")
	events.RegisterShutdownHandler(func(context.Context) error {
		firstCalls++
		events.RegisterShutdownHandler(func(context.Context) error {
			secondCalls++
			return nil
		})
		return wantErr
	})

	done := make(chan error, 1)
	go func() {
		done <- events.TriggerShutdown(context.Background())
	}()

	select {
	case err := <-done:
		if !errors.Is(err, wantErr) {
			t.Fatalf("TriggerShutdown error = %v, want %v", err, wantErr)
		}
	case <-time.After(time.Second):
		t.Fatal("TriggerShutdown held its lock while invoking a handler")
	}

	if firstCalls != 1 || secondCalls != 0 {
		t.Fatalf("first trigger calls=(%d,%d), want=(1,0)", firstCalls, secondCalls)
	}

	if err := events.TriggerShutdown(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("second TriggerShutdown error = %v, want %v", err, wantErr)
	}
	if firstCalls != 2 || secondCalls != 1 {
		t.Fatalf("second trigger calls=(%d,%d), want=(2,1)", firstCalls, secondCalls)
	}
}
