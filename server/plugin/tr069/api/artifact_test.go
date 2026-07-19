package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	artifactResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type artifactAPIFakeStore struct {
	objects map[string][]byte
}

func (s *artifactAPIFakeStore) Begin(context.Context, service.ObjectSpec) (service.ArtifactWriter, error) {
	return nil, errors.New("not implemented")
}

func (s *artifactAPIFakeStore) Open(_ context.Context, key string) (io.ReadCloser, service.ObjectStat, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, service.ObjectStat{}, service.ErrArtifactNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), service.ObjectStat{Key: key, Size: int64(len(data))}, nil
}

func (s *artifactAPIFakeStore) Stat(context.Context, string) (service.ObjectStat, error) {
	return service.ObjectStat{}, errors.New("not implemented")
}

func (s *artifactAPIFakeStore) Delete(context.Context, string) error { return nil }

func TestArtifactAPIListsByExactDeviceAndStreamsSafeDownload(t *testing.T) {
	db := seedArtifactAPI(t)
	store := &artifactAPIFakeStore{objects: map[string][]byte{"private/log/object-1": []byte("base-station-log")}}
	artifactAPI := NewArtifactApi(service.NewTransferStore(db), store)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/tr069/artifact/list", artifactAPI.List)
	router.GET("/tr069/artifact/:fileId/download", artifactAPI.Download)

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/tr069/artifact/list?page=1&pageSize=10&serialNumber=SNB123456789", nil))
	var listResponse struct {
		Code int `json:"code"`
		Data struct {
			List  []artifactResponse.ArtifactSummary `json:"list"`
			Total int64                              `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResponse.Code != 0 || listResponse.Data.Total != 1 || len(listResponse.Data.List) != 1 {
		t.Fatalf("list response = %#v", listResponse)
	}
	if got := listResponse.Data.List[0]; got.FileID == 0 || got.SerialNumber != "SNB123456789" {
		t.Fatalf("filtered item = %#v", got)
	}
	fileID := listResponse.Data.List[0].FileID
	for _, secret := range []string{"private/log/object-1", "sourceIp", "driver", "password", "artifactId", "deviceId", "oui", "status", "sha256"} {
		if strings.Contains(listRecorder.Body.String(), secret) {
			t.Fatalf("list leaked internal field %q: %s", secret, listRecorder.Body.String())
		}
	}
	partialRecorder := httptest.NewRecorder()
	router.ServeHTTP(partialRecorder, httptest.NewRequest(http.MethodGet, "/tr069/artifact/list?page=1&pageSize=10&serialNumber=SNB123", nil))
	if !strings.Contains(partialRecorder.Body.String(), `"total":0`) {
		t.Fatalf("partial serial number matched: %s", partialRecorder.Body.String())
	}

	downloadRecorder := httptest.NewRecorder()
	router.ServeHTTP(downloadRecorder, httptest.NewRequest(http.MethodGet, "/tr069/artifact/"+strconv.FormatUint(fileID, 10)+"/download", nil))
	if downloadRecorder.Code != http.StatusOK || downloadRecorder.Body.String() != "base-station-log" {
		t.Fatalf("download status/body = %d/%q", downloadRecorder.Code, downloadRecorder.Body.String())
	}
	contentDisposition := downloadRecorder.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "bs-log.tar.gz") || strings.ContainsAny(contentDisposition, "\r\n") {
		t.Fatalf("unsafe content disposition %q", contentDisposition)
	}

	missingRecorder := httptest.NewRecorder()
	router.ServeHTTP(missingRecorder, httptest.NewRequest(http.MethodGet, "/tr069/artifact/999999/download", nil))
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("missing artifact status = %d", missingRecorder.Code)
	}
}

func seedArtifactAPI(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.TransferTask), new(model.Artifact)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	devices := []model.Device{
		{SerialNumber: "SNB123456789", OUI: "001122"},
		{SerialNumber: "SNB-SECOND-ABC", OUI: "334455"},
	}
	for index := range devices {
		if err := db.Create(&devices[index]).Error; err != nil {
			t.Fatalf("create device: %v", err)
		}
	}
	now := time.Now().UTC()
	tasks := []model.TransferTask{
		{TaskID: "task-1", DeviceID: devices[0].ID, Channel: "LOG", Source: model.TransferSourcePeriodic, Status: model.TransferStatusCompleted, CreatedAt: now},
		{TaskID: "task-2", DeviceID: devices[1].ID, Channel: "LOG", Source: model.TransferSourcePeriodic, Status: model.TransferStatusCompleted, CreatedAt: now.Add(-time.Minute)},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("create tasks: %v", err)
	}
	receivedAt := now
	artifacts := []model.Artifact{
		{TaskID: "task-1", DeviceID: devices[0].ID, Channel: "LOG", Status: model.ArtifactStatusAvailable, Driver: "minio", ObjectKey: "private/log/object-1", OriginalName: "../../bs-log.tar.gz\r\nX-Test: injected", ContentType: "application/gzip", Size: 16, SHA256: strings.Repeat("a", 64), ReceivedAt: &receivedAt, CreatedAt: now},
		{TaskID: "task-2", DeviceID: devices[1].ID, Channel: "LOG", Status: model.ArtifactStatusAvailable, Driver: "minio", ObjectKey: "private/log/object-2", OriginalName: "other.tar.gz", ContentType: "application/gzip", Size: 8, SHA256: strings.Repeat("b", 64), ReceivedAt: &receivedAt, CreatedAt: now.Add(-time.Minute)},
	}
	if err := db.Create(&artifacts).Error; err != nil {
		t.Fatalf("create artifacts: %v", err)
	}
	return db
}
