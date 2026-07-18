package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type fakeFileRequestAuthenticator struct {
	channel    string
	challenges []string
	err        error
}

func (a fakeFileRequestAuthenticator) Authenticate(*http.Request) (string, []string, error) {
	return a.channel, a.challenges, a.err
}

type fakeUploadDeviceResolver struct {
	device service.UploadDeviceIdentity
	err    error
}

func (r fakeUploadDeviceResolver) Resolve(context.Context, string, string) (service.UploadDeviceIdentity, error) {
	return r.device, r.err
}

type recordingTransferReceiver struct {
	payload []byte
	request service.ReceiveRequest
	err     error
}

func (r *recordingTransferReceiver) Receive(_ context.Context, request service.ReceiveRequest) (model.Artifact, error) {
	r.request = request
	if request.Body != nil {
		r.payload, _ = io.ReadAll(request.Body)
	}
	return model.Artifact{ArtifactID: "artifact-1"}, r.err
}

type staticFileIngressChannelProvider struct {
	channel FileIngressChannel
	ok      bool
}

func (p staticFileIngressChannelProvider) Channel(string) (FileIngressChannel, bool) {
	return p.channel, p.ok
}

type countingRequestBody struct {
	reader io.Reader
	reads  int
}

func (b *countingRequestBody) Read(data []byte) (int, error) {
	b.reads++
	return b.reader.Read(data)
}

func (b *countingRequestBody) Close() error { return nil }

func newFileIngressTestEngine(auth FileRequestAuthenticator, resolver UploadDeviceResolver, receiver TransferReceiveService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	channel := FileIngressChannel{
		Name: "LOG", MaxFileSize: 1024, MaxConcurrent: 4, MaxConcurrentPerDevice: 1,
		UploadTimeout: time.Minute, RetentionDays: 30, StoragePrefix: "artifacts", Driver: "memory",
	}
	engine.Any("/acs/log", NewFileIngressHandler(auth, resolver, receiver, staticFileIngressChannelProvider{channel: channel, ok: true}))
	return engine
}

func TestFileIngressPutAndPostUseSameRawBodyPipeline(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			receiver := new(recordingTransferReceiver)
			engine := newFileIngressTestEngine(
				fakeFileRequestAuthenticator{channel: "LOG"},
				fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
				receiver,
			)
			request := httptest.NewRequest(method, "/acs/log", bytes.NewReader([]byte("raw-log-archive")))
			request.RemoteAddr = "192.0.2.10:1234"
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent || string(receiver.payload) != "raw-log-archive" || receiver.request.Channel != "LOG" {
				t.Fatalf("status=%d payload=%q request=%#v", response.Code, receiver.payload, receiver.request)
			}
		})
	}
}

func TestFileIngressAuthenticatesBeforeReadingBody(t *testing.T) {
	body := &countingRequestBody{reader: bytes.NewReader([]byte("must-not-be-read"))}
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{challenges: []string{`Basic realm="GVA"`, `Digest realm="GVA"`}, err: middleware.ErrFileAuthentication},
		fakeUploadDeviceResolver{}, new(recordingTransferReceiver),
	)
	request := httptest.NewRequest(http.MethodPut, "/acs/log", nil)
	request.Body = body
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || body.reads != 0 || len(response.Header().Values("WWW-Authenticate")) != 2 {
		t.Fatalf("status=%d reads=%d challenges=%#v", response.Code, body.reads, response.Header().Values("WWW-Authenticate"))
	}
}

func TestFileIngressMapsIdentitySizeBusyAndMethodFailures(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		resolverErr error
		receiverErr error
		length      int64
		want        int
	}{
		{name: "unknown", method: http.MethodPut, resolverErr: service.ErrUploadDeviceNotFound, want: http.StatusForbidden},
		{name: "ambiguous", method: http.MethodPost, resolverErr: service.ErrUploadDeviceAmbiguous, want: http.StatusForbidden},
		{name: "declared too large", method: http.MethodPut, length: 2048, want: http.StatusRequestEntityTooLarge},
		{name: "busy", method: http.MethodPost, receiverErr: service.ErrTransferBusy, want: http.StatusServiceUnavailable},
		{name: "method", method: http.MethodDelete, want: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receiver := &recordingTransferReceiver{err: test.receiverErr}
			engine := newFileIngressTestEngine(fakeFileRequestAuthenticator{channel: "LOG"}, fakeUploadDeviceResolver{
				device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}, err: test.resolverErr,
			}, receiver)
			request := httptest.NewRequest(test.method, "/acs/log", bytes.NewReader([]byte("payload")))
			request.RemoteAddr = "192.0.2.10:1234"
			if test.length > 0 {
				request.ContentLength = test.length
			}
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%q", response.Code, test.want, response.Body.String())
			}
			if test.want == http.StatusServiceUnavailable && response.Header().Get("Retry-After") == "" {
				t.Fatal("busy response missing Retry-After")
			}
			if test.want == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "PUT, POST" {
				t.Fatalf("Allow=%q", response.Header().Get("Allow"))
			}
		})
	}
}
