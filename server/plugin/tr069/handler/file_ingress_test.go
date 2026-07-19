package handler

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
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
	if r.err != nil {
		return model.Artifact{}, r.err
	}
	if request.Body != nil {
		r.payload, _ = io.ReadAll(request.Body)
	}
	return model.Artifact{ID: 1}, nil
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
	bytes  int
}

func (b *countingRequestBody) Read(data []byte) (int, error) {
	b.reads++
	n, err := b.reader.Read(data)
	b.bytes += n
	return n, err
}

func (b *countingRequestBody) Close() error { return nil }

func newFileIngressTestEngine(auth FileRequestAuthenticator, resolver UploadDeviceResolver, receiver TransferReceiveService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	channel := FileIngressChannel{
		Name: "LOG", MaxFileSize: 1024, MaxConcurrent: 4, MaxConcurrentPerDevice: 1,
		UploadTimeout: time.Minute, RetentionDays: 30, StoragePrefix: "artifacts", Driver: "memory",
	}
	handler := NewFileIngressHandler(auth, resolver, receiver, staticFileIngressChannelProvider{channel: channel, ok: true})
	engine.Any("/acs/log", handler)
	engine.Any("/acs/log/*filename", handler)
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
			if response.Code != http.StatusCreated || string(receiver.payload) != "raw-log-archive" || receiver.request.Channel != "LOG" {
				t.Fatalf("status=%d payload=%q request=%#v", response.Code, receiver.payload, receiver.request)
			}
		})
	}
}

func TestFileIngressAcceptsVendorPutFilenameSuffix(t *testing.T) {
	receiver := new(recordingTransferReceiver)
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{channel: "LOG"},
		fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
		receiver,
	)
	request := httptest.NewRequest(http.MethodPut, "/acs/log/Log_20260719.tar.gz", bytes.NewReader([]byte("vendor-put-log")))
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if string(receiver.payload) != "vendor-put-log" || receiver.request.OriginalName != "Log_20260719.tar.gz" {
		t.Fatalf("payload=%q originalName=%q", receiver.payload, receiver.request.OriginalName)
	}
}

func TestFileIngressStreamsVendorMultipartPost(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "Log_20260719.tar.gz")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("vendor-post-log")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	receiver := new(recordingTransferReceiver)
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{channel: "LOG"},
		fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
		receiver,
	)
	request := httptest.NewRequest(http.MethodPost, "/acs/log/", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if string(receiver.payload) != "vendor-post-log" || receiver.request.OriginalName != "Log_20260719.tar.gz" {
		t.Fatalf("payload=%q originalName=%q", receiver.payload, receiver.request.OriginalName)
	}
}

func TestFileIngressMultipartBusyDoesNotDrainFileBody(t *testing.T) {
	const boundary = "gva-tr069-boundary"
	prefix := "--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="file"; filename="large.tar.gz"` + "\r\n" +
		"Content-Type: application/gzip\r\n\r\n"
	bodyBytes := []byte(prefix + string(bytes.Repeat([]byte("x"), 128<<10)) + "\r\n--" + boundary + "--\r\n")
	body := &countingRequestBody{reader: bytes.NewReader(bodyBytes)}
	receiver := &recordingTransferReceiver{err: service.ErrTransferBusy}
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{channel: "LOG"},
		fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
		receiver,
	)
	request := httptest.NewRequest(http.MethodPost, "/acs/log/", nil)
	request.Body = body
	request.ContentLength = -1
	request.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
	if body.bytes >= len(bodyBytes)/2 {
		t.Fatalf("busy upload drained %d of %d bytes", body.bytes, len(bodyBytes))
	}
}

func TestFileIngressRejectsMultipartFieldsBeforeFileWithoutDraining(t *testing.T) {
	const boundary = "gva-tr069-boundary"
	bodyBytes := []byte("--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="metadata"` + "\r\n\r\n" +
		string(bytes.Repeat([]byte("x"), 128<<10)) + "\r\n" +
		"--" + boundary + "\r\n" +
		`Content-Disposition: form-data; name="file"; filename="log.tar.gz"` + "\r\n\r\nlog\r\n" +
		"--" + boundary + "--\r\n")
	body := &countingRequestBody{reader: bytes.NewReader(bodyBytes)}
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{channel: "LOG"},
		fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
		new(recordingTransferReceiver),
	)
	request := httptest.NewRequest(http.MethodPost, "/acs/log/", nil)
	request.Body = body
	request.ContentLength = -1
	request.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
	if body.bytes >= len(bodyBytes)/2 {
		t.Fatalf("invalid multipart drained %d of %d bytes", body.bytes, len(bodyBytes))
	}
}

func TestFileIngressRejectsDeclaredOversizeMultipartBeforeReading(t *testing.T) {
	body := &countingRequestBody{reader: bytes.NewReader([]byte("must-not-be-read"))}
	engine := newFileIngressTestEngine(
		fakeFileRequestAuthenticator{channel: "LOG"},
		fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
		new(recordingTransferReceiver),
	)
	request := httptest.NewRequest(http.MethodPost, "/acs/log/", nil)
	request.Body = body
	request.ContentLength = 1024 + (1 << 20) + 1
	request.Header.Set("Content-Type", "multipart/form-data; boundary=oversize")
	request.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge || body.reads != 0 {
		t.Fatalf("status=%d reads=%d", response.Code, body.reads)
	}
}

func TestFileIngressRejectsUnsupportedVendorSuffixForms(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/acs/log/log.tar.gz"},
		{method: http.MethodPut, path: "/acs/log/nested/log.tar.gz"},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			engine := newFileIngressTestEngine(
				fakeFileRequestAuthenticator{channel: "LOG"},
				fakeUploadDeviceResolver{device: service.UploadDeviceIdentity{DeviceID: 1, SerialNumber: "BS-1", OUI: "8CE468"}},
				new(recordingTransferReceiver),
			)
			request := httptest.NewRequest(test.method, test.path, bytes.NewReader([]byte("log")))
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d", response.Code)
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
		{name: "device deleting", method: http.MethodPut, receiverErr: service.ErrDeviceDeleting, want: http.StatusConflict},
		{name: "different retry content", method: http.MethodPut, receiverErr: service.ErrTransferContentConflict, want: http.StatusConflict},
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
