package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioPutResult struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
}

type MinioObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
}

type MinioObjectClient interface {
	PutObject(context.Context, string, string, io.Reader, int64, string, map[string]string) (MinioPutResult, error)
	OpenObject(context.Context, string, string) (io.ReadCloser, MinioObjectInfo, error)
	StatObject(context.Context, string, string) (MinioObjectInfo, error)
	DeleteObject(context.Context, string, string) error
}

type MinioArtifactStore struct {
	client MinioObjectClient
	bucket string
}

func NewMinioArtifactStore(client MinioObjectClient, bucket string) (*MinioArtifactStore, error) {
	if client == nil {
		return nil, errors.New("MinIO object client is required")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, errors.New("MinIO bucket is required")
	}
	return &MinioArtifactStore{client: client, bucket: bucket}, nil
}

func NewMinioArtifactStoreClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioArtifactStore, error) {
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: useSSL})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ensureMinioBucket(ctx, client, bucket); err != nil {
		return nil, fmt.Errorf("initialize MinIO bucket %q: %w", bucket, err)
	}
	return NewMinioArtifactStore(&minioSDKObjectClient{client: client}, bucket)
}

type minioBucketAdmin interface {
	BucketExists(context.Context, string) (bool, error)
	MakeBucket(context.Context, string, minio.MakeBucketOptions) error
}

func ensureMinioBucket(ctx context.Context, client minioBucketAdmin, bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if client == nil || bucket == "" {
		return errors.New("MinIO bucket administrator and bucket are required")
	}
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		if minio.ToErrorResponse(err).Code == "BucketAlreadyOwnedByYou" {
			return nil
		}
		return err
	}
	return nil
}

func (s *MinioArtifactStore) Begin(ctx context.Context, spec service.ObjectSpec) (service.ArtifactWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validMinioObjectKey(spec.Key) {
		return nil, errors.New("invalid MinIO object key")
	}
	uploadCtx, cancel := context.WithCancel(ctx)
	reader, writer := io.Pipe()
	result := make(chan minioUploadResult, 1)
	go func() {
		defer reader.Close()
		info, err := s.client.PutObject(uploadCtx, s.bucket, spec.Key, reader, -1, spec.ContentType, cloneStringMap(spec.Metadata))
		result <- minioUploadResult{info: info, err: err}
	}()
	return &minioArtifactWriter{
		key: spec.Key, pipe: writer, cancel: cancel, result: result, state: minioWriterActive,
	}, nil
}

func (s *MinioArtifactStore) Open(ctx context.Context, key string) (io.ReadCloser, service.ObjectStat, error) {
	reader, info, err := s.client.OpenObject(ctx, s.bucket, key)
	if err != nil {
		return nil, service.ObjectStat{}, normalizeMinioError(err)
	}
	return reader, objectStatFromMinio(info), nil
}

func (s *MinioArtifactStore) Stat(ctx context.Context, key string) (service.ObjectStat, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key)
	if err != nil {
		return service.ObjectStat{}, normalizeMinioError(err)
	}
	return objectStatFromMinio(info), nil
}

func (s *MinioArtifactStore) Delete(ctx context.Context, key string) error {
	return normalizeMinioError(s.client.DeleteObject(ctx, s.bucket, key))
}

type minioWriterState uint8

const (
	minioWriterActive minioWriterState = iota
	minioWriterCommitting
	minioWriterCommitted
	minioWriterAborted
)

type minioUploadResult struct {
	info MinioPutResult
	err  error
}

type minioArtifactWriter struct {
	mu     sync.Mutex
	key    string
	pipe   *io.PipeWriter
	cancel context.CancelFunc
	result <-chan minioUploadResult
	state  minioWriterState
	stat   service.ObjectStat
}

func (w *minioArtifactWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	state := w.state
	w.mu.Unlock()
	switch state {
	case minioWriterAborted:
		return 0, service.ErrArtifactWriterAborted
	case minioWriterCommitted, minioWriterCommitting:
		return 0, service.ErrArtifactWriterCommitted
	default:
		return w.pipe.Write(data)
	}
}

func (w *minioArtifactWriter) Commit(ctx context.Context) (service.ObjectStat, error) {
	w.mu.Lock()
	switch w.state {
	case minioWriterAborted:
		w.mu.Unlock()
		return service.ObjectStat{}, service.ErrArtifactWriterAborted
	case minioWriterCommitted:
		stat := w.stat
		w.mu.Unlock()
		return stat, nil
	case minioWriterCommitting:
		w.mu.Unlock()
		return service.ObjectStat{}, service.ErrArtifactWriterCommitted
	default:
		w.state = minioWriterCommitting
	}
	w.mu.Unlock()

	if err := w.pipe.Close(); err != nil {
		w.fail()
		return service.ObjectStat{}, err
	}
	select {
	case result := <-w.result:
		if result.err != nil {
			w.fail()
			return service.ObjectStat{}, normalizeMinioError(result.err)
		}
		stat := service.ObjectStat{Key: result.info.Key, Size: result.info.Size, ETag: result.info.ETag, LastModified: result.info.LastModified}
		if stat.Key == "" {
			stat.Key = w.key
		}
		w.mu.Lock()
		w.stat = stat
		w.state = minioWriterCommitted
		w.mu.Unlock()
		w.cancel()
		return stat, nil
	case <-ctx.Done():
		w.fail()
		_ = w.pipe.CloseWithError(ctx.Err())
		return service.ObjectStat{}, ctx.Err()
	}
}

func (w *minioArtifactWriter) Abort(ctx context.Context) error {
	w.mu.Lock()
	if w.state == minioWriterAborted || w.state == minioWriterCommitted {
		w.mu.Unlock()
		return nil
	}
	w.state = minioWriterAborted
	w.cancel()
	_ = w.pipe.CloseWithError(service.ErrArtifactWriterAborted)
	w.mu.Unlock()

	select {
	case <-w.result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *minioArtifactWriter) fail() {
	w.mu.Lock()
	w.state = minioWriterAborted
	w.cancel()
	w.mu.Unlock()
}

func validMinioObjectKey(key string) bool {
	return key != "" && !strings.HasPrefix(key, "/") && !strings.Contains(key, "\\") && path.Clean(key) == key && key != "." && key != ".." && !strings.HasPrefix(key, "../")
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func objectStatFromMinio(info MinioObjectInfo) service.ObjectStat {
	return service.ObjectStat{Key: info.Key, Size: info.Size, ETag: info.ETag, ContentType: info.ContentType, LastModified: info.LastModified}
}

func normalizeMinioError(err error) error {
	if err == nil || errors.Is(err, service.ErrArtifactNotFound) {
		return err
	}
	response := minio.ToErrorResponse(err)
	if response.Code == "NoSuchKey" || response.Code == "NoSuchObject" || response.Code == "NoSuchBucket" {
		return service.ErrArtifactNotFound
	}
	return err
}

type minioSDKObjectClient struct {
	client *minio.Client
}

func (c *minioSDKObjectClient) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) (MinioPutResult, error) {
	info, err := c.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType, UserMetadata: metadata})
	return MinioPutResult{Key: info.Key, Size: info.Size, ETag: info.ETag, LastModified: info.LastModified}, err
}

func (c *minioSDKObjectClient) OpenObject(ctx context.Context, bucket, key string) (io.ReadCloser, MinioObjectInfo, error) {
	info, err := c.StatObject(ctx, bucket, key)
	if err != nil {
		return nil, MinioObjectInfo{}, err
	}
	object, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, MinioObjectInfo{}, err
	}
	return object, info, nil
}

func (c *minioSDKObjectClient) StatObject(ctx context.Context, bucket, key string) (MinioObjectInfo, error) {
	info, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	return MinioObjectInfo{Key: info.Key, Size: info.Size, ETag: info.ETag, ContentType: info.ContentType, LastModified: info.LastModified}, err
}

func (c *minioSDKObjectClient) DeleteObject(ctx context.Context, bucket, key string) error {
	return c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}
