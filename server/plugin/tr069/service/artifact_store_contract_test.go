package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type memoryArtifactStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

type memoryArtifactWriter struct {
	store   *memoryArtifactStore
	key     string
	buffer  bytes.Buffer
	aborted bool
	done    bool
}

func newMemoryArtifactStore() *memoryArtifactStore {
	return &memoryArtifactStore{objects: make(map[string][]byte)}
}

func (s *memoryArtifactStore) Begin(ctx context.Context, spec ObjectSpec) (ArtifactWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &memoryArtifactWriter{store: s, key: spec.Key}, nil
}

func (s *memoryArtifactStore) Open(ctx context.Context, key string) (io.ReadCloser, ObjectStat, error) {
	if err := ctx.Err(); err != nil {
		return nil, ObjectStat{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, ObjectStat{}, ErrArtifactNotFound
	}
	copyData := append([]byte(nil), data...)
	return io.NopCloser(bytes.NewReader(copyData)), ObjectStat{Key: key, Size: int64(len(copyData))}, nil
}

func (s *memoryArtifactStore) Stat(ctx context.Context, key string) (ObjectStat, error) {
	reader, stat, err := s.Open(ctx, key)
	if reader != nil {
		_ = reader.Close()
	}
	return stat, err
}

func (s *memoryArtifactStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.objects[key]; !ok {
		return ErrArtifactNotFound
	}
	delete(s.objects, key)
	return nil
}

func (w *memoryArtifactWriter) Write(data []byte) (int, error) {
	if w.aborted {
		return 0, ErrArtifactWriterAborted
	}
	if w.done {
		return 0, ErrArtifactWriterCommitted
	}
	return w.buffer.Write(data)
}

func (w *memoryArtifactWriter) Commit(ctx context.Context) (ObjectStat, error) {
	if err := ctx.Err(); err != nil {
		return ObjectStat{}, err
	}
	if w.aborted {
		return ObjectStat{}, ErrArtifactWriterAborted
	}
	if !w.done {
		w.store.mu.Lock()
		w.store.objects[w.key] = append([]byte(nil), w.buffer.Bytes()...)
		w.store.mu.Unlock()
		w.done = true
	}
	return ObjectStat{Key: w.key, Size: int64(w.buffer.Len())}, nil
}

func (w *memoryArtifactWriter) Abort(context.Context) error {
	w.aborted = true
	return nil
}

func runArtifactStoreContract(t *testing.T, store ArtifactStore) {
	t.Helper()
	ctx := context.Background()
	key := "artifacts/log/1/2026/07/19/artifact-1"
	w, err := store.Begin(ctx, ObjectSpec{Key: key, ContentType: "application/gzip"})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := w.Write([]byte("first-")); err != nil {
		t.Fatalf("write first chunk: %v", err)
	}
	if _, err := w.Write([]byte("second")); err != nil {
		t.Fatalf("write second chunk: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("object visible before commit: %v", err)
	}
	stat, err := w.Commit(ctx)
	if err != nil || stat.Key != key || stat.Size != int64(len("first-second")) {
		t.Fatalf("commit stat=%#v err=%v", stat, err)
	}
	reader, opened, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "first-second" || opened.Size != stat.Size {
		t.Fatalf("data=%q opened=%#v err=%v", data, opened, err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("stat after delete: %v", err)
	}

	abortWriter, err := store.Begin(ctx, ObjectSpec{Key: key + "-abort"})
	if err != nil {
		t.Fatalf("begin abort writer: %v", err)
	}
	if err := abortWriter.Abort(ctx); err != nil {
		t.Fatalf("abort: %v", err)
	}
	if err := abortWriter.Abort(ctx); err != nil {
		t.Fatalf("second abort: %v", err)
	}
	if _, err := abortWriter.Commit(ctx); !errors.Is(err, ErrArtifactWriterAborted) {
		t.Fatalf("commit after abort: %v", err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Begin(canceled, ObjectSpec{Key: key + "-canceled"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("begin with canceled context: %v", err)
	}
}

func TestArtifactStoreContract(t *testing.T) {
	runArtifactStoreContract(t, newMemoryArtifactStore())
}

func TestArtifactObjectKeyAndOriginalNameSanitization(t *testing.T) {
	key, err := ArtifactObjectKey("artifacts", "LOG", 42, time.Date(2026, 7, 19, 4, 5, 6, 0, time.FixedZone("local", 8*60*60)), "artifact-id")
	if err != nil {
		t.Fatalf("object key: %v", err)
	}
	if key != "artifacts/log/42/2026/07/18/artifact-id" {
		t.Fatalf("key=%q", key)
	}
	if _, err := ArtifactObjectKey("artifacts", "LOG", 42, time.Now(), "../escape"); err == nil {
		t.Fatal("path traversal artifact ID accepted")
	}
	if got := SanitizeArtifactOriginalName("../bad\x00\nname.tar.gz"); got != "badname.tar.gz" {
		t.Fatalf("sanitized name=%q", got)
	}
}
