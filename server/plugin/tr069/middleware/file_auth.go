package middleware

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

var (
	ErrFileAuthentication    = errors.New("file ingress authentication failed")
	ErrFileAuthConfiguration = errors.New("file ingress authentication configuration invalid")
)

type FileCredential struct {
	Channel  string
	Username string
	Password string
	Realm    string
	Schemes  []string
	NonceTTL time.Duration
}

type CredentialProvider interface {
	CredentialForRequest(*http.Request) (FileCredential, error)
}

type CredentialProviderFunc func(*http.Request) (FileCredential, error)

func (f CredentialProviderFunc) CredentialForRequest(request *http.Request) (FileCredential, error) {
	return f(request)
}

type DigestNonceStore interface {
	Issue(context.Context, time.Duration) (string, error)
	Consume(context.Context, string, string, string, uint32) (bool, error)
}

type FileAuthenticator struct {
	credentials CredentialProvider
	nonces      DigestNonceStore
}

func NewFileAuthenticator(credentials CredentialProvider, nonces DigestNonceStore) *FileAuthenticator {
	return &FileAuthenticator{credentials: credentials, nonces: nonces}
}

func (a *FileAuthenticator) Authenticate(request *http.Request) (string, []string, error) {
	if a == nil || a.credentials == nil || request == nil {
		return "", nil, ErrFileAuthConfiguration
	}
	credential, err := a.credentials.CredentialForRequest(request)
	if err != nil {
		return "", nil, errors.Join(ErrFileAuthConfiguration, err)
	}
	if err := validateFileCredential(credential); err != nil {
		return "", nil, err
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorization == "" {
		return "", a.challenges(request.Context(), credential), ErrFileAuthentication
	}
	scheme, payload, found := strings.Cut(authorization, " ")
	if !found || !schemeAllowed(credential.Schemes, scheme) {
		return "", a.challenges(request.Context(), credential), ErrFileAuthentication
	}
	var authenticated bool
	switch strings.ToLower(scheme) {
	case "basic":
		authenticated = verifyBasicCredential(payload, credential)
	case "digest":
		authenticated = a.verifyDigest(request, payload, credential)
	}
	if !authenticated {
		return "", a.challenges(request.Context(), credential), ErrFileAuthentication
	}
	return credential.Channel, nil, nil
}

func validateFileCredential(credential FileCredential) error {
	if strings.TrimSpace(credential.Channel) == "" || credential.Username == "" || credential.Password == "" || credential.Realm == "" {
		return ErrFileAuthConfiguration
	}
	if len(credential.Schemes) == 0 {
		return ErrFileAuthConfiguration
	}
	for _, scheme := range credential.Schemes {
		if !strings.EqualFold(scheme, "basic") && !strings.EqualFold(scheme, "digest") {
			return ErrFileAuthConfiguration
		}
		if strings.EqualFold(scheme, "digest") && credential.NonceTTL <= 0 {
			return ErrFileAuthConfiguration
		}
	}
	return nil
}

func (a *FileAuthenticator) challenges(ctx context.Context, credential FileCredential) []string {
	challenges := make([]string, 0, len(credential.Schemes))
	for _, scheme := range credential.Schemes {
		switch strings.ToLower(scheme) {
		case "basic":
			challenges = append(challenges, fmt.Sprintf(`Basic realm="%s"`, quoteAuthValue(credential.Realm)))
		case "digest":
			if a.nonces == nil {
				continue
			}
			nonce, err := a.nonces.Issue(ctx, credential.NonceTTL)
			if err == nil && nonce != "" {
				challenges = append(challenges, fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5, qop="auth"`, quoteAuthValue(credential.Realm), quoteAuthValue(nonce)))
			}
		}
	}
	return challenges
}

func verifyBasicCredential(payload string, credential FileCredential) bool {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
	if err != nil {
		return false
	}
	username, password, found := strings.Cut(string(decoded), ":")
	return found && constantTimeStringEqual(username, credential.Username) && constantTimeStringEqual(password, credential.Password)
}

func (a *FileAuthenticator) verifyDigest(request *http.Request, payload string, credential FileCredential) bool {
	if a.nonces == nil {
		return false
	}
	fields, err := parseDigestFields(payload)
	if err != nil {
		return false
	}
	username := fields["username"]
	realm := fields["realm"]
	nonce := fields["nonce"]
	uri := fields["uri"]
	response := fields["response"]
	qop := strings.ToLower(fields["qop"])
	nonceCountText := fields["nc"]
	cnonce := fields["cnonce"]
	algorithm := fields["algorithm"]
	if algorithm == "" {
		algorithm = "MD5"
	}
	if !constantTimeStringEqual(username, credential.Username) || !constantTimeStringEqual(realm, credential.Realm) ||
		uri != request.URL.RequestURI() || qop != "auth" || nonce == "" || cnonce == "" || len(nonceCountText) != 8 ||
		(!strings.EqualFold(algorithm, "MD5") && !strings.EqualFold(algorithm, "MD5-sess")) {
		return false
	}
	nonceCount64, err := strconv.ParseUint(nonceCountText, 16, 32)
	if err != nil || nonceCount64 == 0 {
		return false
	}
	ha1 := digestMD5Hex(credential.Username + ":" + credential.Realm + ":" + credential.Password)
	if strings.EqualFold(algorithm, "MD5-sess") {
		ha1 = digestMD5Hex(ha1 + ":" + nonce + ":" + cnonce)
	}
	ha2 := digestMD5Hex(request.Method + ":" + uri)
	expected := digestMD5Hex(ha1 + ":" + nonce + ":" + nonceCountText + ":" + cnonce + ":auth:" + ha2)
	if !constantTimeStringEqual(strings.ToLower(response), expected) {
		return false
	}
	consumed, err := a.nonces.Consume(request.Context(), nonce, credential.Username, cnonce, uint32(nonceCount64))
	return err == nil && consumed
}

func parseDigestFields(payload string) (map[string]string, error) {
	parts := splitAuthFields(payload)
	fields := make(map[string]string, len(parts))
	for _, part := range parts {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found || strings.TrimSpace(key) == "" {
			return nil, ErrFileAuthentication
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, `"`) {
			if len(value) < 2 || !strings.HasSuffix(value, `"`) {
				return nil, ErrFileAuthentication
			}
			value = strings.ReplaceAll(strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`), `\\`, `\`)
		}
		if _, exists := fields[key]; exists {
			return nil, ErrFileAuthentication
		}
		fields[key] = value
	}
	return fields, nil
}

func splitAuthFields(value string) []string {
	var parts []string
	start := 0
	quoted := false
	escaped := false
	for index, char := range value {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quoted {
			escaped = true
			continue
		}
		if char == '"' {
			quoted = !quoted
			continue
		}
		if char == ',' && !quoted {
			parts = append(parts, value[start:index])
			start = index + 1
		}
	}
	parts = append(parts, value[start:])
	return parts
}

func schemeAllowed(schemes []string, target string) bool {
	for _, scheme := range schemes {
		if strings.EqualFold(scheme, target) {
			return true
		}
	}
	return false
}

func constantTimeStringEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func digestMD5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func quoteAuthValue(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`)
}

type RuntimeFileCredentialProvider struct{}

func (RuntimeFileCredentialProvider) CredentialForRequest(request *http.Request) (FileCredential, error) {
	if request == nil {
		return FileCredential{}, ErrFileAuthConfiguration
	}
	runtime := config.CurrentRuntime()
	for name, channel := range runtime.Settings.FileIngress.Channels {
		if channel.Enabled && channel.Path == request.URL.Path {
			return FileCredential{
				Channel: strings.ToUpper(name), Username: runtime.Settings.FileIngress.Authentication.Username,
				Password: runtime.Settings.FileIngress.Authentication.Password, Realm: runtime.Settings.FileIngress.Authentication.Realm,
				Schemes: append([]string(nil), runtime.Settings.FileIngress.Authentication.Schemes...), NonceTTL: runtime.FileIngress.NonceTTL,
			}, nil
		}
	}
	return FileCredential{}, ErrFileAuthConfiguration
}
