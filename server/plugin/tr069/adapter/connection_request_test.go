package adapter

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
)

func testDigestFields(header string) map[string]string {
	fields := make(map[string]string)
	header = strings.TrimSpace(strings.TrimPrefix(header, "Digest"))
	for _, part := range strings.Split(header, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			continue
		}
		fields[strings.ToLower(pair[0])] = strings.Trim(strings.TrimSpace(pair[1]), `"`)
	}
	return fields
}

func testDigestHex(algorithm, value string) string {
	if strings.EqualFold(algorithm, "SHA-256") {
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	}
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func testDigestResponseMatches(r *http.Request, username, password, algorithm string) bool {
	fields := testDigestFields(r.Header.Get("Authorization"))
	if fields["username"] != username || !strings.EqualFold(fields["algorithm"], algorithm) || fields["qop"] != "auth" {
		return false
	}
	ha1 := testDigestHex(algorithm, username+":"+fields["realm"]+":"+password)
	ha2 := testDigestHex(algorithm, r.Method+":"+fields["uri"])
	want := testDigestHex(algorithm, strings.Join([]string{
		ha1, fields["nonce"], fields["nc"], fields["cnonce"], "auth", ha2,
	}, ":"))
	return fields["response"] == want && fields["uri"] == r.URL.RequestURI()
}

func newDigestConnectionRequestServer(t *testing.T, username, password, algorithm string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("first Authorization = %q, want empty", got)
			}
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm="cpe", nonce="abc", qop="auth", algorithm=%s`, algorithm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !testDigestResponseMatches(r, username, password, algorithm) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	return server, &calls
}

func useConnectionRequestAllowedCIDRs(t *testing.T, allowedCIDRs []string) {
	t.Helper()
	previous := tr069config.CurrentRuntime()
	connectionRequestTransport.CloseIdleConnections()
	t.Cleanup(func() {
		connectionRequestTransport.CloseIdleConnections()
		tr069config.StoreRuntime(previous.Settings)
	})
	settings := previous.Settings
	settings.ConnectionRequest.AllowedCIDRs = append([]string(nil), allowedCIDRs...)
	tr069config.StoreRuntime(settings)
}

func allowLoopbackConnectionRequests(t *testing.T) {
	t.Helper()
	useConnectionRequestAllowedCIDRs(t, []string{"127.0.0.0/8"})
}

func TestDoConnectionRequestUsesDigestAndReusesConnection(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	server, calls := newDigestConnectionRequestServer(t, "acs", "secret", "MD5")
	var connections atomic.Int32
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	status, err := doConnectionRequest(context.Background(), server.URL+"/wake?source=test", "acs", "secret", time.Second)
	if err != nil || status != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("status=%d calls=%d err=%v", status, calls.Load(), err)
	}
	if connections.Load() != 1 {
		t.Fatalf("TCP connections = %d, want one reused connection", connections.Load())
	}
}

func TestDoConnectionRequestSupportsSHA256Digest(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	server, calls := newDigestConnectionRequestServer(t, "sha-user", "sha-secret", "SHA-256")
	server.Start()
	defer server.Close()

	status, err := doConnectionRequest(context.Background(), server.URL+"/wake", "sha-user", "sha-secret", time.Second)
	if err != nil || status != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("status=%d calls=%d err=%v", status, calls.Load(), err)
	}
}

func TestDoConnectionRequestPinsOneDNSResolutionAcrossClosedDigestConnection(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	var calls atomic.Int32
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if !strings.HasPrefix(r.Host, "cpe.example:") {
			t.Errorf("Host = %q, want original cpe.example host", r.Host)
		}
		if call == 1 {
			w.Header().Set("Connection", "close")
			w.Header().Set("WWW-Authenticate", `Digest realm="cpe", nonce="abc", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !testDigestResponseMatches(r, "acs", "secret", "MD5") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	var lookups atomic.Int32
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		if lookups.Add(1) == 1 {
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("192.0.2.10")}, nil
	}
	port := server.Listener.Addr().(*net.TCPAddr).Port
	status, err := doConnectionRequestWithLookup(
		context.Background(), fmt.Sprintf("http://cpe.example:%d/wake", port), "acs", "secret", time.Second, lookup,
	)
	if err != nil || status != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("status=%d calls=%d err=%v", status, calls.Load(), err)
	}
	if lookups.Load() != 1 {
		t.Fatalf("DNS lookups = %d, want exactly one for both Digest requests", lookups.Load())
	}
	if connections.Load() != 2 {
		t.Fatalf("TCP connections = %d, want two pinned connections after Connection: close", connections.Load())
	}
}

func TestDoConnectionRequestRejectsWrongDigestCredentials(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	server, calls := newDigestConnectionRequestServer(t, "acs", "correct-secret", "MD5")
	server.Start()
	defer server.Close()

	status, err := doConnectionRequest(context.Background(), server.URL, "acs", "wrong-secret", time.Second)
	if status != http.StatusUnauthorized || err == nil || calls.Load() != 2 {
		t.Fatalf("status=%d calls=%d err=%v, want two calls and 401", status, calls.Load(), err)
	}
	if strings.Contains(err.Error(), "wrong-secret") {
		t.Fatalf("error leaked password: %v", err)
	}
}

func TestDoConnectionRequestRejectsRedirect(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()

	status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second)
	if status != http.StatusFound || err == nil {
		t.Fatalf("status=%d err=%v, want redirect rejection", status, err)
	}
	if targetCalls.Load() != 0 {
		t.Fatalf("redirect target calls = %d, want zero", targetCalls.Load())
	}
}

func TestDoConnectionRequestRejectsURLUserinfoWithoutLeakingIt(t *testing.T) {
	status, err := doConnectionRequest(context.Background(), "http://alice:super-secret@127.0.0.1:9/wake", "", "", time.Second)
	if status != 0 || err == nil {
		t.Fatalf("status=%d err=%v, want URL userinfo rejection", status, err)
	}
	if strings.Contains(err.Error(), "alice") || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("userinfo leaked in error: %v", err)
	}
}

func TestDoConnectionRequestRejectsDisallowedCIDR(t *testing.T) {
	previous := tr069config.CurrentRuntime()
	t.Cleanup(func() { tr069config.StoreRuntime(previous.Settings) })
	settings := previous.Settings
	settings.ConnectionRequest.AllowedCIDRs = []string{"192.0.2.0/24"}
	tr069config.StoreRuntime(settings)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second)
	if status != 0 || err == nil || !strings.Contains(strings.ToLower(err.Error()), "cidr") {
		t.Fatalf("status=%d err=%v, want CIDR rejection", status, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("server calls = %d, want no network request", calls.Load())
	}
}

func TestDoConnectionRequestRejectsEmptyAllowedCIDRsBeforeNetwork(t *testing.T) {
	useConnectionRequestAllowedCIDRs(t, nil)
	var calls atomic.Int32
	var lookups atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	port := server.Listener.Addr().(*net.TCPAddr).Port
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		lookups.Add(1)
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}

	status, err := doConnectionRequestWithLookup(
		context.Background(), fmt.Sprintf("http://cpe.example:%d/wake", port), "", "", time.Second, lookup,
	)
	if status != 0 || err == nil || !strings.Contains(strings.ToLower(err.Error()), "cidr") {
		t.Fatalf("status=%d err=%v, want missing CIDR policy rejection", status, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("server calls = %d, want no network request", calls.Load())
	}
	if lookups.Load() != 0 {
		t.Fatalf("DNS lookups = %d, want fail-closed before resolution", lookups.Load())
	}
}

func TestDoConnectionRequestHonorsTimeout(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	started := time.Now()
	status, err := doConnectionRequest(context.Background(), server.URL, "", "", 10*time.Millisecond)
	if status != 0 || err == nil || time.Since(started) >= 90*time.Millisecond {
		t.Fatalf("status=%d elapsed=%s err=%v, want bounded timeout", status, time.Since(started), err)
	}
}

func TestWriteConnectionRequestLogOmitsAllURLAndSecretMaterial(t *testing.T) {
	previousRuntime := tr069config.CurrentRuntime()
	settings := previousRuntime.Settings
	settings.InfoLogEnable = true
	settings.InfoLogDir = t.TempDir()
	tr069config.StoreRuntime(settings)
	previousLogger := global.GVA_LOG
	core, observed := observer.New(zap.DebugLevel)
	global.GVA_LOG = zap.New(core)
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
		tr069config.StoreRuntime(previousRuntime.Settings)
	})

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = oldStdout })

	writeConnectionRequestLog(ConnectionRequestResult{
		URL:     "http://alice:super-secret@example.com/wake?token=hidden",
		Elapsed: 25 * time.Millisecond,
		Err:     errors.New("Get http://example.com/wake?token=hidden Authorization: Digest password=super-secret"),
	}, 7, 0)
	_ = writer.Close()
	os.Stdout = oldStdout
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	_ = reader.Close()
	infoPath := filepath.Join(settings.InfoLogDir, time.Now().Format("2006-01-02"), "tr069info.log")
	infoOutput, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("read infolog: %v", err)
	}
	var zapOutput strings.Builder
	for _, entry := range observed.All() {
		zapOutput.WriteString(entry.Message)
		zapOutput.WriteString(fmt.Sprint(entry.ContextMap()))
	}
	combined := string(output) + string(infoOutput) + zapOutput.String()
	for _, forbidden := range []string{
		"http://", "example.com", "/wake", "token=hidden", "alice", "super-secret", "Authorization", "Digest", "password=",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("observability leaked %q: %q", forbidden, combined)
		}
	}
	for _, required := range []string{"deviceId", "7", "elapsed", "25ms", "connection_request_failed"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("safe log field %q missing: %q", required, combined)
		}
	}
}

func TestDigestAuthorizationRejectsUnsupportedAlgorithmAndQOP(t *testing.T) {
	for name, challenge := range map[string]string{
		"algorithm": `Digest realm="cpe", nonce="abc", qop="auth", algorithm=MD5-sess`,
		"qop":       `Digest realm="cpe", nonce="abc", qop="auth-int", algorithm=MD5`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := digestAuthorization(challenge, http.MethodGet, "/wake", "acs", "secret")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), name) {
				t.Fatalf("digestAuthorization() error = %v, want explicit %s rejection", err, name)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("digest error leaked password: %v", err)
			}
		})
	}
}

func TestValidateConnectionRequestURLAcceptsHTTPAndHTTPS(t *testing.T) {
	for _, raw := range []string{"http://192.0.2.1:7547/wake", "https://cpe.example/wake"} {
		parsed, err := validateConnectionRequestURL(raw)
		if err != nil || parsed.String() != raw {
			t.Fatalf("validateConnectionRequestURL(%q) = (%v, %v)", raw, parsed, err)
		}
	}
	for _, raw := range []string{"ftp://cpe.example/wake", "http://alice:secret@cpe.example/wake", "http:///wake"} {
		if _, err := validateConnectionRequestURL(raw); err == nil {
			t.Fatalf("validateConnectionRequestURL(%q) succeeded", raw)
		} else if strings.Contains(err.Error(), "alice") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("URL validation leaked userinfo: %v", err)
		}
	}
}

type sizedReadCloser struct {
	remaining int64
	read      int64
	closed    bool
}

func (r *sizedReadCloser) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > r.remaining {
		n = r.remaining
	}
	for i := int64(0); i < n; i++ {
		p[i] = 'x'
	}
	r.remaining -= n
	r.read += n
	return int(n), nil
}

func (r *sizedReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestConnectionRequestResponseBodyDrainIsBounded(t *testing.T) {
	body := &sizedReadCloser{remaining: 4 * maxConnectionRequestResponseBytes}
	drainAndCloseConnectionRequestBody(body)
	if !body.closed {
		t.Fatal("response body was not closed")
	}
	if body.read > maxConnectionRequestResponseBytes {
		t.Fatalf("response body bytes read = %d, cap = %d", body.read, maxConnectionRequestResponseBytes)
	}
}

func TestResolveConnectionRequestAddressPinsOneAllowedIP(t *testing.T) {
	var lookups int
	lookup := func(_ context.Context, network, host string) ([]netip.Addr, error) {
		lookups++
		if network != "ip" || host != "cpe.example" {
			t.Fatalf("lookup = (%q, %q)", network, host)
		}
		return []netip.Addr{netip.MustParseAddr("10.20.30.40"), netip.MustParseAddr("10.20.30.41")}, nil
	}
	address, err := resolveConnectionRequestAddress(
		context.Background(), "tcp", "cpe.example:7547", []string{"10.0.0.0/8"}, lookup,
	)
	if err != nil || address != "10.20.30.40:7547" {
		t.Fatalf("resolved address = %q, err=%v", address, err)
	}
	if lookups != 1 {
		t.Fatalf("DNS lookups = %d, want one", lookups)
	}
}

func TestResolveConnectionRequestAddressRejectsEveryDisallowedIP(t *testing.T) {
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.20.30.40")}, nil
	}
	_, err := resolveConnectionRequestAddress(
		context.Background(), "tcp", "cpe.example:7547", []string{"192.0.2.0/24"}, lookup,
	)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "cidr") {
		t.Fatalf("resolve error = %v, want CIDR rejection", err)
	}
}

func TestConnectionRequestSharedTransportReusesAcrossCalls(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	for i := 0; i < 2; i++ {
		if status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second); err != nil || status != http.StatusNoContent {
			t.Fatalf("call %d = (%d, %v)", i, status, err)
		}
	}
	if connections.Load() != 1 {
		t.Fatalf("TCP connections = %d, want shared transport reuse", connections.Load())
	}
}

func TestConnectionRequestCIDRPolicyRevocationDoesNotReuseAllowedConnection(t *testing.T) {
	previous := tr069config.CurrentRuntime()
	connectionRequestTransport.CloseIdleConnections()
	t.Cleanup(func() {
		connectionRequestTransport.CloseIdleConnections()
		tr069config.StoreRuntime(previous.Settings)
	})
	settings := previous.Settings
	settings.ConnectionRequest.AllowedCIDRs = []string{"127.0.0.0/8"}
	tr069config.StoreRuntime(settings)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second); err != nil || status != http.StatusNoContent {
		t.Fatalf("allowed request = (%d, %v)", status, err)
	}

	settings.ConnectionRequest.AllowedCIDRs = []string{"192.0.2.0/24"}
	tr069config.StoreRuntime(settings)
	if status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second); status != 0 || err == nil {
		t.Fatalf("request after CIDR reload = (%d, %v), want rejection", status, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("server calls after policy revocation = %d, want old connection unused", calls.Load())
	}
}

func TestConnectionRequestTransportPoolIsVersionedByCIDRPolicy(t *testing.T) {
	previous := tr069config.CurrentRuntime()
	connectionRequestTransport.CloseIdleConnections()
	t.Cleanup(func() {
		connectionRequestTransport.CloseIdleConnections()
		tr069config.StoreRuntime(previous.Settings)
	})
	settings := previous.Settings
	settings.ConnectionRequest.AllowedCIDRs = []string{"127.0.0.0/8"}
	tr069config.StoreRuntime(settings)

	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()

	if status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second); err != nil || status != http.StatusNoContent {
		t.Fatalf("first policy request = (%d, %v)", status, err)
	}
	settings.ConnectionRequest.AllowedCIDRs = []string{"127.0.0.1/32"}
	tr069config.StoreRuntime(settings)
	if status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second); err != nil || status != http.StatusNoContent {
		t.Fatalf("second policy request = (%d, %v)", status, err)
	}
	if connections.Load() != 2 {
		t.Fatalf("TCP connections = %d, want distinct pools for distinct CIDR policies", connections.Load())
	}
}

func TestConnectionRequestTransportPoolIsIsolatedByPinnedIP(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	firstListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen first pinned IP: %v", err)
	}
	port := firstListener.Addr().(*net.TCPAddr).Port
	secondListener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.2:%d", port))
	if err != nil {
		_ = firstListener.Close()
		t.Fatalf("listen second pinned IP: %v", err)
	}
	var firstCalls atomic.Int32
	firstServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		firstCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	_ = firstServer.Listener.Close()
	firstServer.Listener = firstListener
	firstServer.Start()
	defer firstServer.Close()
	var secondCalls atomic.Int32
	secondServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secondCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	_ = secondServer.Listener.Close()
	secondServer.Listener = secondListener
	secondServer.Start()
	defer secondServer.Close()

	requestURL := fmt.Sprintf("http://cpe.example:%d/wake", port)
	for _, pinned := range []string{"127.0.0.1", "127.0.0.2"} {
		lookup := func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr(pinned)}, nil
		}
		status, err := doConnectionRequestWithLookup(context.Background(), requestURL, "", "", time.Second, lookup)
		if err != nil || status != http.StatusNoContent {
			t.Fatalf("request pinned to %s = (%d, %v)", pinned, status, err)
		}
	}
	if firstCalls.Load() != 1 || secondCalls.Load() != 1 {
		t.Fatalf("pinned server calls = %d/%d, want 1/1", firstCalls.Load(), secondCalls.Load())
	}
}

func TestConnectionRequestPinnedTLSRoutePreservesOriginalSNI(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	sni := make(chan string, 1)
	var handlerCalls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalls.Add(1)
	}))
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			select {
			case sni <- hello.ServerName:
			default:
			}
			return nil, nil
		},
	}
	server.StartTLS()
	defer server.Close()
	port := server.Listener.Addr().(*net.TCPAddr).Port
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	status, err := doConnectionRequestWithLookup(
		context.Background(), fmt.Sprintf("https://cpe.example:%d/wake", port), "", "", time.Second, lookup,
	)
	if status != 0 || err == nil {
		t.Fatalf("TLS request = (%d, %v), want certificate failure after ClientHello", status, err)
	}
	if handlerCalls.Load() != 0 {
		t.Fatalf("TLS handler calls = %d, want certificate rejection before HTTP", handlerCalls.Load())
	}
	select {
	case got := <-sni:
		if got != "cpe.example" {
			t.Fatalf("TLS SNI = %q, want cpe.example", got)
		}
	case <-time.After(time.Second):
		t.Fatal("TLS server did not observe ClientHello")
	}
}

func connectionRequestTestCredentialConfig() tr069config.ConnectionRequestConfig {
	return tr069config.ConnectionRequestConfig{
		CredentialKeyVersion:    "wake-v1",
		CredentialEncryptionKey: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32)),
		RequestTimeout:          2,
		AllowedCIDRs:            []string{"127.0.0.0/8"},
		AuthScheme:              "digest",
	}
}

func newConnectionRequestProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open profile test DB: %v", err)
	}
	if err := db.AutoMigrate(new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate connection profile: %v", err)
	}
	return db
}

func seedResolvedConnectionRequestProfile(t *testing.T, db *gorm.DB, deviceID uint, rawURL, username, password string, settings tr069config.ConnectionRequestConfig) model.ConnectionProfile {
	t.Helper()
	cipher, err := NewCredentialCipher(settings)
	if err != nil {
		t.Fatalf("new profile cipher: %v", err)
	}
	encrypted, err := cipher.Encrypt(password)
	if err != nil {
		t.Fatalf("encrypt profile password: %v", err)
	}
	profile := model.ConnectionProfile{
		DeviceID:             deviceID,
		DiscoveredURL:        rawURL,
		Username:             username,
		PasswordCiphertext:   encrypted.Ciphertext,
		CredentialKeyVersion: encrypted.Version,
		CredentialSource:     model.ConnectionCredentialSourceManual,
		AuthScheme:           "digest",
		ProvisionState:       model.ConnectionProfileStateReady,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed connection profile: %v", err)
	}
	return profile
}

func useConnectionRequestRuntimeAndDB(t *testing.T, db *gorm.DB, settings tr069config.ConnectionRequestConfig) {
	t.Helper()
	previousRuntime := tr069config.CurrentRuntime()
	previousDB := global.GVA_DB
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		tr069config.StoreRuntime(previousRuntime.Settings)
	})
	global.GVA_DB = db
	runtimeSettings := previousRuntime.Settings
	runtimeSettings.ConnectionRequest = settings
	tr069config.StoreRuntime(runtimeSettings)
}

func TestTriggerConnectionRequestResolvesOneProfileAndRecordsWakeSummaryOnly(t *testing.T) {
	server, calls := newDigestConnectionRequestServer(t, "profile-user", "profile-secret", "MD5")
	server.Start()
	defer server.Close()

	db := newConnectionRequestProfileTestDB(t)
	settings := connectionRequestTestCredentialConfig()
	useConnectionRequestRuntimeAndDB(t, db, settings)
	seeded := seedResolvedConnectionRequestProfile(t, db, 42, server.URL+"/wake", "profile-user", "profile-secret", settings)

	var profileQueries atomic.Int32
	if err := db.Callback().Query().Before("gorm:query").Register("test:count-profile-resolve", func(tx *gorm.DB) {
		if tx.Statement.Table == (model.ConnectionProfile{}).TableName() {
			profileQueries.Add(1)
		}
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}

	result := TriggerConnectionRequest(context.Background(), 42, ConnectionRequestConfig{})
	if result.Err != nil || result.StatusCode != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("TriggerConnectionRequest() = %#v calls=%d", result, calls.Load())
	}
	if profileQueries.Load() != 1 {
		t.Fatalf("profile SELECT queries = %d, want one Resolve query", profileQueries.Load())
	}

	var updated model.ConnectionProfile
	if err := db.First(&updated, "device_id = ?", 42).Error; err != nil {
		t.Fatalf("reload connection profile: %v", err)
	}
	if updated.LastWakeAt == nil || updated.LastWakeStatus != connectionRequestWakeSucceeded || updated.LastError != "" {
		t.Fatalf("wake summary = at:%v status:%q error:%q", updated.LastWakeAt, updated.LastWakeStatus, updated.LastError)
	}
	if updated.DiscoveredURL != seeded.DiscoveredURL || updated.Username != seeded.Username ||
		!bytes.Equal(updated.PasswordCiphertext, seeded.PasswordCiphertext) || updated.ProvisionState != seeded.ProvisionState {
		t.Fatalf("wake summary modified profile inputs: before=%#v after=%#v", seeded, updated)
	}
	if !updated.UpdatedAt.Equal(seeded.UpdatedAt) {
		t.Fatalf("wake summary changed UpdatedAt: before=%s after=%s", seeded.UpdatedAt, updated.UpdatedAt)
	}
}

func TestTriggerConnectionRequestUsesHotReloadedRuntimeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(1500 * time.Millisecond)
	}))
	defer server.Close()

	db := newConnectionRequestProfileTestDB(t)
	settings := connectionRequestTestCredentialConfig()
	settings.RequestTimeout = 1
	useConnectionRequestRuntimeAndDB(t, db, settings)
	seedResolvedConnectionRequestProfile(t, db, 43, server.URL, "profile-user", "profile-secret", settings)

	started := time.Now()
	result := TriggerConnectionRequest(context.Background(), 43, ConnectionRequestConfig{Timeout: 5 * time.Second})
	elapsed := time.Since(started)
	if result.Err == nil || elapsed >= 1400*time.Millisecond {
		t.Fatalf("TriggerConnectionRequest() elapsed=%s result=%#v, want runtime one-second timeout", elapsed, result)
	}
	if strings.Contains(result.Err.Error(), "profile-secret") {
		t.Fatalf("timeout result leaked password: %v", result.Err)
	}
}

func TestDoConnectionRequestRejectsNon2xx(t *testing.T) {
	allowLoopbackConnectionRequests(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	status, err := doConnectionRequest(context.Background(), server.URL, "", "", time.Second)
	if status != http.StatusUnauthorized || err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("doConnectionRequest() = (%d, %v), want 401 error", status, err)
	}
}
