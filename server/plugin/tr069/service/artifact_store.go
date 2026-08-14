package service

import (
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrArtifactNotFound        = errors.New("artifact object not found")
	ErrArtifactWriterAborted   = errors.New("artifact writer aborted")
	ErrArtifactWriterCommitted = errors.New("artifact writer already committed")
)

type ObjectSpec struct {
	Key         string
	ContentType string
	Metadata    map[string]string
}

type ObjectStat struct {
	Key          string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
}

type ArtifactStore interface {
	Begin(context.Context, ObjectSpec) (ArtifactWriter, error)
	Open(context.Context, string) (io.ReadCloser, ObjectStat, error)
	Stat(context.Context, string) (ObjectStat, error)
	Delete(context.Context, string) error
}

type ArtifactWriter interface {
	io.Writer
	Commit(context.Context) (ObjectStat, error)
	// Abort may be called concurrently with Write or Commit when device
	// deletion cancels an active upload. Implementations must make it safe
	// and idempotent.
	Abort(context.Context) error
}

func ArtifactObjectKey(prefix, channel string, deviceID uint, receivedAt time.Time, fileID uint64) (string, error) {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" || !isSafeObjectPath(prefix) {
		return "", errors.New("invalid artifact storage prefix")
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !isSafeObjectSegment(channel) {
		return "", errors.New("invalid artifact channel")
	}
	if deviceID == 0 {
		return "", errors.New("artifact device ID is required")
	}
	if fileID == 0 {
		return "", errors.New("file ID is required")
	}
	if receivedAt.IsZero() {
		return "", errors.New("artifact receive time is required")
	}
	return path.Join(prefix, channel, strconv.FormatUint(uint64(deviceID), 10), receivedAt.UTC().Format("2006/01/02"), strconv.FormatUint(fileID, 10)), nil
}

func isSafeObjectPath(value string) bool {
	if strings.Contains(value, "\\") || path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if !isSafeObjectSegment(segment) {
			return false
		}
	}
	return true
}

func isSafeObjectSegment(value string) bool {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\") {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func SanitizeArtifactOriginalName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if unicode.IsControl(char) {
			continue
		}
		charBytes := utf8.RuneLen(char)
		if charBytes < 0 || builder.Len()+charBytes > 255 {
			break
		}
		builder.WriteRune(char)
	}
	name := strings.TrimSpace(builder.String())
	if name == "." || name == ".." {
		return ""
	}
	return name
}
