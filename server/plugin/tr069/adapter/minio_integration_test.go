package adapter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/google/uuid"
)

func TestMinioArtifactStoreIntegration(t *testing.T) {
	if os.Getenv("TR069_MINIO_INTEGRATION") != "1" {
		t.Skip("set TR069_MINIO_INTEGRATION=1 to run against an isolated MinIO")
	}
	endpoint := envOr("TR069_MINIO_ENDPOINT", "127.0.0.1:19000")
	accessKey := envOr("TR069_MINIO_ACCESS_KEY", "gva-minio")
	secretKey := envOr("TR069_MINIO_SECRET_KEY", "gva-minio-change-me")
	bucket := envOr("TR069_MINIO_BUCKET", "gva-tr069-artifacts-test")
	store, err := NewMinioArtifactStoreClient(endpoint, accessKey, secretKey, bucket, false)
	if err != nil {
		t.Fatalf("create store and bootstrap bucket: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	key := "integration/log/" + uuid.NewString()
	payload := bytes.Repeat([]byte("tr069-log-smoke\n"), 1024)
	writer, err := store.Begin(ctx, service.ObjectSpec{Key: key, ContentType: "application/gzip"})
	if err != nil {
		t.Fatalf("begin upload: %v", err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("stream upload: %v", err)
	}
	stat, err := writer.Commit(ctx)
	if err != nil {
		t.Fatalf("commit upload: %v", err)
	}
	if stat.Size != int64(len(payload)) {
		t.Fatalf("stored size = %d, want %d", stat.Size, len(payload))
	}

	reader, opened, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open object: %v", err)
	}
	received, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read object: read=%v close=%v", readErr, closeErr)
	}
	if opened.Size != int64(len(payload)) || !bytes.Equal(received, payload) {
		t.Fatalf("downloaded object differs: size=%d bytes=%d", opened.Size, len(received))
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, service.ErrArtifactNotFound) {
		t.Fatalf("stat deleted object = %v, want ErrArtifactNotFound", err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
