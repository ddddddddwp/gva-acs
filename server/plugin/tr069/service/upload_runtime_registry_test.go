package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type registryArtifactWriter struct {
	aborted atomic.Bool
}

func (w *registryArtifactWriter) Write(value []byte) (int, error) { return len(value), nil }
func (w *registryArtifactWriter) Commit(context.Context) (ObjectStat, error) {
	return ObjectStat{}, nil
}
func (w *registryArtifactWriter) Abort(context.Context) error {
	w.aborted.Store(true)
	return nil
}

func TestUploadRuntimeRegistryBlocksCancelsAndReleasesDevice(t *testing.T) {
	registry := NewUploadRuntimeRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	handle, err := registry.Register(41, cancel)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	writer := new(registryArtifactWriter)
	if err := handle.AttachWriter(writer); err != nil {
		t.Fatalf("AttachWriter() error = %v", err)
	}

	if err := registry.BlockAndCancel(context.Background(), 41); err != nil {
		t.Fatalf("BlockAndCancel() error = %v", err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("upload context was not cancelled")
	}
	if !writer.aborted.Load() {
		t.Fatal("artifact writer was not aborted")
	}
	if _, err := registry.Register(41, func() {}); !errors.Is(err, ErrDeviceDeleting) {
		t.Fatalf("blocked Register() error = %v, want ErrDeviceDeleting", err)
	}

	registry.Release(41)
	second, err := registry.Register(41, func() {})
	if err != nil {
		t.Fatalf("Register() after Release error = %v", err)
	}
	second.Unregister()
	handle.Unregister()
}

func TestUploadRuntimeRegistryAbortsWriterAttachedAfterBlock(t *testing.T) {
	registry := NewUploadRuntimeRegistry()
	handle, err := registry.Register(52, func() {})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.BlockAndCancel(context.Background(), 52); err != nil {
		t.Fatalf("BlockAndCancel() error = %v", err)
	}

	writer := new(registryArtifactWriter)
	if err := handle.AttachWriter(writer); !errors.Is(err, ErrDeviceDeleting) {
		t.Fatalf("AttachWriter() error = %v, want ErrDeviceDeleting", err)
	}
	if !writer.aborted.Load() {
		t.Fatal("late writer was not aborted")
	}
	handle.Unregister()
}
