package adapter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
)

type fakeMinioObjectClient struct {
	mu       sync.Mutex
	objects  map[string][]byte
	lastSize int64
}

func newFakeMinioObjectClient() *fakeMinioObjectClient {
	return &fakeMinioObjectClient{objects: make(map[string][]byte)}
}

func (f *fakeMinioObjectClient) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (MinioPutResult, error) {
	f.mu.Lock()
	f.lastSize = size
	f.mu.Unlock()
	var buffer bytes.Buffer
	chunk := make([]byte, 7)
	for {
		if err := ctx.Err(); err != nil {
			return MinioPutResult{}, err
		}
		read, err := reader.Read(chunk)
		if read > 0 {
			_, _ = buffer.Write(chunk[:read])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return MinioPutResult{}, err
		}
	}
	f.mu.Lock()
	f.objects[bucket+"/"+key] = append([]byte(nil), buffer.Bytes()...)
	f.mu.Unlock()
	return MinioPutResult{Key: key, Size: int64(buffer.Len()), ETag: "etag", LastModified: time.Now().UTC()}, nil
}

func (f *fakeMinioObjectClient) OpenObject(ctx context.Context, bucket, key string) (io.ReadCloser, MinioObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, MinioObjectInfo{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[bucket+"/"+key]
	if !ok {
		return nil, MinioObjectInfo{}, service.ErrArtifactNotFound
	}
	copyData := append([]byte(nil), data...)
	return io.NopCloser(bytes.NewReader(copyData)), MinioObjectInfo{Key: key, Size: int64(len(copyData)), ETag: "etag"}, nil
}

func (f *fakeMinioObjectClient) StatObject(ctx context.Context, bucket, key string) (MinioObjectInfo, error) {
	reader, stat, err := f.OpenObject(ctx, bucket, key)
	if reader != nil {
		_ = reader.Close()
	}
	return stat, err
}

func (f *fakeMinioObjectClient) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.objects[bucket+"/"+key]; !ok {
		return service.ErrArtifactNotFound
	}
	delete(f.objects, bucket+"/"+key)
	return nil
}

func TestMinioArtifactStoreStreamsAndCommits(t *testing.T) {
	client := newFakeMinioObjectClient()
	store, err := NewMinioArtifactStore(client, "tr069-artifacts")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	key := "artifacts/log/1/2026/07/19/artifact-1"
	w, err := store.Begin(ctx, service.ObjectSpec{Key: key, ContentType: "application/gzip"})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := w.Write([]byte("streamed-")); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if _, err := w.Write([]byte("payload")); err != nil {
		t.Fatalf("write second: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, service.ErrArtifactNotFound) {
		t.Fatalf("object visible before commit: %v", err)
	}
	stat, err := w.Commit(ctx)
	if err != nil || stat.Size != int64(len("streamed-payload")) || stat.Key != key {
		t.Fatalf("commit stat=%#v err=%v", stat, err)
	}
	client.mu.Lock()
	lastSize := client.lastSize
	client.mu.Unlock()
	if lastSize != -1 {
		t.Fatalf("PutObject size=%d, want streaming size -1", lastSize)
	}
	reader, _, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "streamed-payload" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestMinioArtifactWriterAbortIsIdempotent(t *testing.T) {
	store, err := NewMinioArtifactStore(newFakeMinioObjectClient(), "tr069-artifacts")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	w, err := store.Begin(context.Background(), service.ObjectSpec{Key: "artifacts/log/abort"})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := w.Abort(context.Background()); err != nil {
		t.Fatalf("abort: %v", err)
	}
	if err := w.Abort(context.Background()); err != nil {
		t.Fatalf("second abort: %v", err)
	}
	if _, err := w.Commit(context.Background()); !errors.Is(err, service.ErrArtifactWriterAborted) {
		t.Fatalf("commit after abort: %v", err)
	}
}
