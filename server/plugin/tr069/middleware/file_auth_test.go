package middleware

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

type fakeDigestNonceStore struct {
	nonce  string
	counts map[string]uint32
	valid  bool
}

func (s *fakeDigestNonceStore) Issue(context.Context, time.Duration) (string, error) {
	return s.nonce, nil
}

func (s *fakeDigestNonceStore) Consume(_ context.Context, nonce, username, cnonce string, count uint32) (bool, error) {
	if !s.valid || nonce != s.nonce {
		return false, nil
	}
	key := nonce + "\x00" + username + "\x00" + cnonce
	if count <= s.counts[key] {
		return false, nil
	}
	s.counts[key] = count
	return true, nil
}

func testFileCredentialProvider() CredentialProvider {
	return CredentialProviderFunc(func(*http.Request) (FileCredential, error) {
		return FileCredential{
			Channel: "LOG", Username: "log-user", Password: "super-secret", Realm: "GVA-TR069-LOG",
			Schemes: []string{"digest", "basic"}, NonceTTL: 5 * time.Minute,
		}, nil
	})
}

func TestFileAuthBasicSuccessAndFailure(t *testing.T) {
	auth := NewFileAuthenticator(testFileCredentialProvider(), &fakeDigestNonceStore{nonce: "nonce-1", counts: make(map[string]uint32), valid: true})
	request, _ := http.NewRequest(http.MethodPut, "http://example.com/acs/log", nil)
	request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("log-user:super-secret")))
	channel, challenges, err := auth.Authenticate(request)
	if err != nil || channel != "LOG" || len(challenges) != 0 {
		t.Fatalf("channel=%q challenges=%#v err=%v", channel, challenges, err)
	}

	request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("log-user:wrong-secret")))
	_, challenges, err = auth.Authenticate(request)
	if !errors.Is(err, ErrFileAuthentication) || len(challenges) != 2 {
		t.Fatalf("challenges=%#v err=%v", challenges, err)
	}
	combined := err.Error() + strings.Join(challenges, " ")
	if strings.Contains(combined, "super-secret") || strings.Contains(combined, request.Header.Get("Authorization")) {
		t.Fatalf("authentication secret leaked: %s", combined)
	}
}

func TestFileAuthDigestIsMethodSensitiveAndReplaySafe(t *testing.T) {
	nonces := &fakeDigestNonceStore{nonce: "nonce-2", counts: make(map[string]uint32), valid: true}
	auth := NewFileAuthenticator(testFileCredentialProvider(), nonces)
	request, _ := http.NewRequest(http.MethodPut, "http://example.com/acs/log?source=bs", nil)
	authorization := digestAuthorization(http.MethodPut, request.URL.RequestURI(), "MD5", "nonce-2", "00000001", "client-nonce")
	request.Header.Set("Authorization", authorization)
	channel, _, err := auth.Authenticate(request)
	if err != nil || channel != "LOG" {
		t.Fatalf("PUT digest channel=%q err=%v", channel, err)
	}

	post, _ := http.NewRequest(http.MethodPost, request.URL.String(), nil)
	post.Header.Set("Authorization", authorization)
	if _, _, err := auth.Authenticate(post); !errors.Is(err, ErrFileAuthentication) {
		t.Fatalf("PUT digest accepted for POST: %v", err)
	}

	replay, _ := http.NewRequest(http.MethodPut, request.URL.String(), nil)
	replay.Header.Set("Authorization", authorization)
	if _, _, err := auth.Authenticate(replay); !errors.Is(err, ErrFileAuthentication) {
		t.Fatalf("replayed nonce count accepted: %v", err)
	}
}

func TestFileAuthDigestSupportsMD5SessAndRejectsBadFields(t *testing.T) {
	nonces := &fakeDigestNonceStore{nonce: "nonce-3", counts: make(map[string]uint32), valid: true}
	auth := NewFileAuthenticator(testFileCredentialProvider(), nonces)
	request, _ := http.NewRequest(http.MethodPost, "http://example.com/acs/log", nil)
	request.Header.Set("Authorization", digestAuthorization(http.MethodPost, "/acs/log", "MD5-sess", "nonce-3", "00000001", "sess-nonce"))
	if _, _, err := auth.Authenticate(request); err != nil {
		t.Fatalf("MD5-sess: %v", err)
	}

	cases := []string{
		strings.Replace(digestAuthorization(http.MethodPost, "/acs/log", "MD5", "nonce-3", "00000002", "bad-realm"), `realm="GVA-TR069-LOG"`, `realm="wrong"`, 1),
		strings.Replace(digestAuthorization(http.MethodPost, "/wrong", "MD5", "nonce-3", "00000003", "bad-uri"), `uri="/wrong"`, `uri="/different"`, 1),
		strings.Replace(digestAuthorization(http.MethodPost, "/acs/log", "MD5", "nonce-3", "00000004", "bad-response"), `response="`, `response="00`, 1),
	}
	for index, header := range cases {
		request.Header.Set("Authorization", header)
		if _, _, err := auth.Authenticate(request); !errors.Is(err, ErrFileAuthentication) {
			t.Fatalf("case %d accepted: %v", index, err)
		}
	}
}

func TestFileAuthMissingAndEmptyConfiguredCredentials(t *testing.T) {
	auth := NewFileAuthenticator(testFileCredentialProvider(), &fakeDigestNonceStore{nonce: "nonce-4", counts: make(map[string]uint32), valid: true})
	request, _ := http.NewRequest(http.MethodPut, "http://example.com/acs/log", nil)
	if _, challenges, err := auth.Authenticate(request); !errors.Is(err, ErrFileAuthentication) || len(challenges) != 2 {
		t.Fatalf("missing auth challenges=%#v err=%v", challenges, err)
	}

	empty := NewFileAuthenticator(CredentialProviderFunc(func(*http.Request) (FileCredential, error) {
		return FileCredential{Channel: "LOG", Username: "", Password: "", Realm: "GVA", Schemes: []string{"basic"}}, nil
	}), nil)
	if _, _, err := empty.Authenticate(request); !errors.Is(err, ErrFileAuthConfiguration) {
		t.Fatalf("empty configured credentials error=%v", err)
	}
}

func TestRuntimeFileCredentialProviderMatchesVendorPathVariants(t *testing.T) {
	previous := config.CurrentRuntime()
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })
	config.StoreRuntime(config.TR069Config{FileIngress: config.FileIngressConfig{
		Authentication: config.FileIngressAuthConfig{
			Username: "log-user", Password: "super-secret", Realm: "GVA-TR069-LOG", Schemes: []string{"basic"},
		},
		Channels: map[string]config.TransferChannelConfig{
			"log": {Enabled: true, Path: "/acs/log"},
		},
	}})

	provider := RuntimeFileCredentialProvider{}
	for _, target := range []string{"/acs/log", "/acs/log/", "/acs/log/Log_20260719.tar.gz"} {
		request, err := http.NewRequest(http.MethodPut, "http://example.com"+target, nil)
		if err != nil {
			t.Fatalf("new request %s: %v", target, err)
		}
		credential, err := provider.CredentialForRequest(request)
		if err != nil || credential.Channel != "LOG" {
			t.Fatalf("target=%s channel=%q err=%v", target, credential.Channel, err)
		}
	}

	rejected, _ := http.NewRequest(http.MethodPut, "http://example.com/acs/logger", nil)
	if _, err := provider.CredentialForRequest(rejected); !errors.Is(err, ErrFileAuthConfiguration) {
		t.Fatalf("unrelated path accepted: %v", err)
	}
	for _, rejectedRequest := range []*http.Request{
		func() *http.Request {
			request, _ := http.NewRequest(http.MethodPost, "http://example.com/acs/log/log.tar.gz", nil)
			return request
		}(),
		func() *http.Request {
			request, _ := http.NewRequest(http.MethodPut, "http://example.com/acs/log/nested/log.tar.gz", nil)
			return request
		}(),
	} {
		if _, err := provider.CredentialForRequest(rejectedRequest); !errors.Is(err, ErrFileAuthConfiguration) {
			t.Fatalf("unsupported path accepted: %s %s", rejectedRequest.Method, rejectedRequest.URL.Path)
		}
	}
}

func digestAuthorization(method, uri, algorithm, nonce, nc, cnonce string) string {
	username, realm, password := "log-user", "GVA-TR069-LOG", "super-secret"
	ha1 := md5Hex(username + ":" + realm + ":" + password)
	if strings.EqualFold(algorithm, "MD5-sess") {
		ha1 = md5Hex(ha1 + ":" + nonce + ":" + cnonce)
	}
	ha2 := md5Hex(method + ":" + uri)
	response := md5Hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm=%s, response="%s", qop=auth, nc=%s, cnonce="%s"`, username, realm, nonce, uri, algorithm, response, nc, cnonce)
}

func md5Hex(value string) string {
	digest := md5.Sum([]byte(value))
	return hex.EncodeToString(digest[:])
}
