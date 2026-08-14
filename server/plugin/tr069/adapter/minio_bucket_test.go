package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
)

type fakeMinioBucketAdmin struct {
	exists    bool
	existsErr error
	makeErr   error
	created   []string
}

func (f *fakeMinioBucketAdmin) BucketExists(context.Context, string) (bool, error) {
	return f.exists, f.existsErr
}

func (f *fakeMinioBucketAdmin) MakeBucket(_ context.Context, bucket string, _ minio.MakeBucketOptions) error {
	f.created = append(f.created, bucket)
	return f.makeErr
}

func TestEnsureMinioBucketCreatesMissingBucketAndIsRaceSafe(t *testing.T) {
	missing := new(fakeMinioBucketAdmin)
	if err := ensureMinioBucket(context.Background(), missing, "gva-tr069-artifacts"); err != nil {
		t.Fatalf("create missing bucket: %v", err)
	}
	if len(missing.created) != 1 || missing.created[0] != "gva-tr069-artifacts" {
		t.Fatalf("created buckets = %#v", missing.created)
	}

	existing := &fakeMinioBucketAdmin{exists: true}
	if err := ensureMinioBucket(context.Background(), existing, "gva-tr069-artifacts"); err != nil || len(existing.created) != 0 {
		t.Fatalf("existing bucket result: err=%v created=%#v", err, existing.created)
	}

	raced := &fakeMinioBucketAdmin{makeErr: minio.ErrorResponse{Code: "BucketAlreadyOwnedByYou"}}
	if err := ensureMinioBucket(context.Background(), raced, "gva-tr069-artifacts"); err != nil {
		t.Fatalf("concurrent bucket creation should be accepted: %v", err)
	}

	failure := &fakeMinioBucketAdmin{existsErr: errors.New("unavailable")}
	if err := ensureMinioBucket(context.Background(), failure, "gva-tr069-artifacts"); err == nil {
		t.Fatal("expected bucket probe failure")
	}
}
