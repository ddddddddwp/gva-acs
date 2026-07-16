package adapter

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/infolog"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
)

const (
	maxConnectionRequestResponseBytes = 64 << 10
	connectionRequestWakeSucceeded    = "SUCCESS"
	connectionRequestWakeFailed       = "FAILED"
)

var errConnectionRequestRedirect = errors.New("connection request redirects are disabled")

type connectionRequestLookupFunc func(context.Context, string, string) ([]netip.Addr, error)

type connectionRequestRoute struct {
	origin        string
	pinnedAddress string
	policyKey     string
}

type connectionRequestRouteContextKey struct{}

var connectionRequestDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

type connectionRequestRoutingTransport struct {
	mu         sync.Mutex
	transports map[string]*http.Transport
}

var connectionRequestTransport = &connectionRequestRoutingTransport{
	transports: make(map[string]*http.Transport),
}

var connectionRequestClient = &http.Client{
	Transport: connectionRequestTransport,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return errConnectionRequestRedirect
	},
}

func (t *connectionRequestRoutingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t == nil || request == nil {
		return nil, errors.New("connection request transport is unavailable")
	}
	route, ok := request.Context().Value(connectionRequestRouteContextKey{}).(connectionRequestRoute)
	if !ok || route.origin == "" || route.pinnedAddress == "" || route.policyKey == "" {
		return nil, errors.New("connection request route is unavailable")
	}
	origin, err := normalizedConnectionRequestOrigin(request.URL)
	if err != nil || origin != route.origin {
		return nil, errors.New("connection request route origin mismatch")
	}
	return t.transportFor(route).RoundTrip(request)
}

func (t *connectionRequestRoutingTransport) transportFor(route connectionRequestRoute) *http.Transport {
	key := route.origin + "\x00" + route.pinnedAddress + "\x00" + route.policyKey
	t.mu.Lock()
	defer t.mu.Unlock()
	if transport := t.transports[key]; transport != nil {
		return transport
	}
	pinnedAddress := route.pinnedAddress
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return connectionRequestDialer.DialContext(ctx, network, pinnedAddress)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	t.transports[key] = transport
	return transport
}

func (t *connectionRequestRoutingTransport) CloseIdleConnections() {
	if t == nil {
		return
	}
	t.mu.Lock()
	transports := make([]*http.Transport, 0, len(t.transports))
	for _, transport := range t.transports {
		transports = append(transports, transport)
	}
	t.mu.Unlock()
	for _, transport := range transports {
		transport.CloseIdleConnections()
	}
}

type ConnectionRequestResult struct {
	URL        string
	StatusCode int
	Elapsed    time.Duration
	Err        error
}

type ConnectionRequestConfig struct {
	Timeout time.Duration
	Retries int
}

func TriggerConnectionRequest(ctx context.Context, deviceID uint, cfg ConnectionRequestConfig) ConnectionRequestResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}

	runtime := tr069config.CurrentRuntime()
	profile, err := resolveConnectionRequestTarget(ctx, deviceID, runtime.Settings.ConnectionRequest)
	if err != nil {
		result := ConnectionRequestResult{Err: fmt.Errorf("resolve connection request profile for deviceId=%d: %w", deviceID, err)}
		recordConnectionRequestWake(ctx, deviceID, result)
		return result
	}
	if profile.AuthScheme != "" && !strings.EqualFold(profile.AuthScheme, "digest") {
		result := ConnectionRequestResult{
			URL: profile.URL,
			Err: fmt.Errorf("unsupported connection request authentication scheme for deviceId=%d", deviceID),
		}
		recordConnectionRequestWake(ctx, deviceID, result)
		return result
	}

	timeout := runtime.ConnectionRequestTimeout
	if timeout <= 0 {
		timeout = cfg.Timeout
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	var result ConnectionRequestResult
	for attempt := 0; attempt <= cfg.Retries; attempt++ {
		start := time.Now()
		status, requestErr := doConnectionRequest(ctx, profile.URL, profile.Username, profile.Password, timeout)
		result = ConnectionRequestResult{
			URL:        profile.URL,
			StatusCode: status,
			Elapsed:    time.Since(start),
			Err:        requestErr,
		}
		writeConnectionRequestLog(result, deviceID, attempt)
		if requestErr == nil {
			break
		}
	}
	if summaryErr := recordConnectionRequestWake(ctx, deviceID, result); summaryErr != nil {
		result.Err = errors.Join(result.Err, summaryErr)
	}
	return result
}

func resolveConnectionRequestTarget(ctx context.Context, deviceID uint, settings tr069config.ConnectionRequestConfig) (ResolvedConnectionProfile, error) {
	if global.GVA_DB == nil {
		return ResolvedConnectionProfile{}, errors.New("connection profile database is not initialized")
	}
	cipher, err := NewCredentialCipher(settings)
	if err != nil {
		return ResolvedConnectionProfile{}, err
	}
	return NewConnectionProfileRepository(global.GVA_DB, cipher).Resolve(ctx, deviceID)
}

// connectionRequestTarget preserves the legacy internal lookup shape while resolving
// exactly one Connection Profile row instead of querying the device and data-model tables.
func connectionRequestTarget(ctx context.Context, deviceID uint) (connURL string, user string, pass string, ok bool) {
	profile, err := resolveConnectionRequestTarget(ctx, deviceID, tr069config.CurrentRuntime().Settings.ConnectionRequest)
	if err != nil {
		return "", "", "", false
	}
	return profile.URL, profile.Username, profile.Password, true
}

func recordConnectionRequestWake(ctx context.Context, deviceID uint, result ConnectionRequestResult) error {
	if global.GVA_DB == nil || deviceID == 0 {
		return nil
	}
	status := connectionRequestWakeSucceeded
	lastError := ""
	if result.Err != nil {
		status = connectionRequestWakeFailed
		lastError = boundedConnectionRequestError(result.Err)
	}
	now := time.Now()
	return global.GVA_DB.WithContext(ctx).Model(&model.ConnectionProfile{}).
		Where("device_id = ?", deviceID).
		UpdateColumns(map[string]any{
			"last_wake_at":     &now,
			"last_wake_status": status,
			"last_error":       lastError,
		}).Error
}

func boundedConnectionRequestError(err error) string {
	return connectionRequestErrorCategory(err)
}

func connectionRequestErrorCategory(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, errConnectionRequestRedirect) {
		return "redirect_rejected"
	}
	return "connection_request_failed"
}

func doConnectionRequest(ctx context.Context, connURL string, user string, pass string, timeout time.Duration) (int, error) {
	return doConnectionRequestWithLookup(ctx, connURL, user, pass, timeout, net.DefaultResolver.LookupNetIP)
}

func doConnectionRequestWithLookup(ctx context.Context, connURL string, user string, pass string, timeout time.Duration, lookup connectionRequestLookupFunc) (int, error) {
	parsed, err := validateConnectionRequestURL(connURL)
	if err != nil {
		return 0, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	route, err := resolveConnectionRequestRoute(
		requestCtx, parsed, tr069config.CurrentRuntime().Settings.ConnectionRequest.AllowedCIDRs, lookup,
	)
	if err != nil {
		return 0, err
	}
	requestCtx = context.WithValue(requestCtx, connectionRequestRouteContextKey{}, route)

	first, err := newConnectionRequest(requestCtx, parsed, "")
	if err != nil {
		return 0, err
	}
	response, err := connectionRequestClient.Do(first)
	if err != nil {
		return connectionRequestHTTPFailure(response, err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		return finishConnectionRequestResponse(response)
	}

	challenge := findDigestChallenge(response.Header.Values("WWW-Authenticate"))
	drainAndCloseConnectionRequestBody(response.Body)
	if challenge == "" {
		return http.StatusUnauthorized, connectionRequestStatusError(http.StatusUnauthorized)
	}
	if user == "" {
		return http.StatusUnauthorized, errors.New("connection request digest credentials are required")
	}
	authorization, err := digestAuthorization(challenge, http.MethodGet, parsed.RequestURI(), user, pass)
	if err != nil {
		return http.StatusUnauthorized, err
	}
	authenticated, err := newConnectionRequest(requestCtx, parsed, authorization)
	if err != nil {
		return 0, err
	}
	response, err = connectionRequestClient.Do(authenticated)
	if err != nil {
		return connectionRequestHTTPFailure(response, err)
	}
	return finishConnectionRequestResponse(response)
}

func newConnectionRequest(ctx context.Context, parsed *url.URL, authorization string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, errors.New("create connection request")
	}
	request.Header.Set("User-Agent", "gva-tr069-acs/connection-request")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	return request, nil
}

func finishConnectionRequestResponse(response *http.Response) (int, error) {
	if response == nil {
		return 0, errors.New("connection request returned no response")
	}
	status := response.StatusCode
	drainAndCloseConnectionRequestBody(response.Body)
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return status, connectionRequestStatusError(status)
	}
	return status, nil
}

func connectionRequestHTTPFailure(response *http.Response, err error) (int, error) {
	status := 0
	if response != nil {
		status = response.StatusCode
		if response.Body != nil {
			drainAndCloseConnectionRequestBody(response.Body)
		}
	}
	if errors.Is(err, errConnectionRequestRedirect) {
		return status, errConnectionRequestRedirect
	}
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return status, fmt.Errorf("connection request %s failed: %w", urlError.Op, urlError.Err)
	}
	return status, fmt.Errorf("connection request failed: %w", err)
}

func connectionRequestStatusError(status int) error {
	return fmt.Errorf("connection request returned HTTP status %d", status)
}

func drainAndCloseConnectionRequestBody(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.CopyN(io.Discard, body, maxConnectionRequestResponseBytes)
	_ = body.Close()
}

func validateConnectionRequestURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.IsAbs() == false || parsed.Opaque != "" {
		return nil, errors.New("connection request URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("connection request URL must use http or https")
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return nil, errors.New("connection request URL host is required")
	}
	if parsed.User != nil {
		return nil, errors.New("connection request URL must not contain credentials")
	}
	parsed.Fragment = ""
	return parsed, nil
}

func normalizedConnectionRequestOrigin(parsed *url.URL) (string, error) {
	if parsed == nil {
		return "", errors.New("connection request URL is invalid")
	}
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return "", errors.New("connection request URL scheme is invalid")
		}
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", errors.New("connection request URL host is required")
	}
	return parsed.Scheme + "://" + net.JoinHostPort(host, port), nil
}

func resolveConnectionRequestRoute(ctx context.Context, parsed *url.URL, allowedCIDRs []string, lookup connectionRequestLookupFunc) (connectionRequestRoute, error) {
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return connectionRequestRoute{}, errors.New("connection request URL scheme is invalid")
		}
	}
	host := strings.ToLower(parsed.Hostname())
	origin, err := normalizedConnectionRequestOrigin(parsed)
	if err != nil {
		return connectionRequestRoute{}, err
	}
	prefixes, policyKey, err := connectionRequestCIDRPolicy(allowedCIDRs)
	if err != nil {
		return connectionRequestRoute{}, err
	}
	pinned, err := resolveConnectionRequestAddressWithPrefixes(ctx, "tcp", net.JoinHostPort(host, port), prefixes, lookup)
	if err != nil {
		return connectionRequestRoute{}, err
	}
	return connectionRequestRoute{
		origin:        origin,
		pinnedAddress: pinned,
		policyKey:     policyKey,
	}, nil
}

func resolveConnectionRequestAddress(ctx context.Context, network, address string, allowedCIDRs []string, lookup connectionRequestLookupFunc) (string, error) {
	prefixes, _, err := connectionRequestCIDRPolicy(allowedCIDRs)
	if err != nil {
		return "", err
	}
	return resolveConnectionRequestAddressWithPrefixes(ctx, network, address, prefixes, lookup)
}

func resolveConnectionRequestAddressWithPrefixes(ctx context.Context, network, address string, prefixes []netip.Prefix, lookup connectionRequestLookupFunc) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return "", errors.New("connection request network address is invalid")
	}

	var addresses []netip.Addr
	if literal, parseErr := netip.ParseAddr(host); parseErr == nil {
		addresses = []netip.Addr{literal}
	} else {
		if lookup == nil {
			return "", errors.New("connection request DNS resolver is unavailable")
		}
		addresses, err = lookup(ctx, "ip", host)
		if err != nil {
			return "", fmt.Errorf("connection request DNS lookup failed: %w", err)
		}
	}
	for _, candidate := range addresses {
		candidate = candidate.Unmap()
		if !connectionRequestAddressMatchesNetwork(candidate, network) || !connectionRequestIPAllowed(candidate, prefixes) {
			continue
		}
		return net.JoinHostPort(candidate.String(), port), nil
	}
	return "", errors.New("connection request address is outside allowed CIDRs")
}

func connectionRequestCIDRPolicy(values []string) ([]netip.Prefix, string, error) {
	prefixes, err := parseConnectionRequestCIDRs(values)
	if err != nil {
		return nil, "", err
	}
	canonical := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		canonical = append(canonical, prefix.String())
	}
	sort.Strings(canonical)
	return prefixes, strings.Join(canonical, ","), nil
}

func parseConnectionRequestCIDRs(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("invalid connection request allowed CIDR %q", value)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	if len(prefixes) == 0 {
		return nil, errors.New("connection request allowed CIDRs are required")
	}
	return prefixes, nil
}

func connectionRequestAddressMatchesNetwork(address netip.Addr, network string) bool {
	switch network {
	case "tcp4":
		return address.Is4()
	case "tcp6":
		return address.Is6()
	default:
		return true
	}
}

func connectionRequestIPAllowed(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func findDigestChallenge(values []string) string {
	for _, value := range values {
		lower := strings.ToLower(value)
		if index := strings.Index(lower, "digest "); index >= 0 {
			return strings.TrimSpace(value[index:])
		}
	}
	return ""
}

func digestAuthorization(challenge, method, requestURI, username, password string) (string, error) {
	parameters, err := parseDigestChallenge(challenge)
	if err != nil {
		return "", err
	}
	realm := parameters["realm"]
	nonce := parameters["nonce"]
	if realm == "" || nonce == "" {
		return "", errors.New("digest challenge is missing realm or nonce")
	}
	algorithm := parameters["algorithm"]
	if algorithm == "" {
		algorithm = "MD5"
	}
	if !strings.EqualFold(algorithm, "MD5") && !strings.EqualFold(algorithm, "SHA-256") {
		return "", errors.New("unsupported digest algorithm")
	}
	algorithm = strings.ToUpper(algorithm)
	if algorithm == "SHA-256" {
		algorithm = "SHA-256"
	}
	if !digestQOPSupportsAuth(parameters["qop"]) {
		return "", errors.New("unsupported digest qop")
	}

	cnonceBytes := make([]byte, 16)
	if _, err := rand.Read(cnonceBytes); err != nil {
		return "", errors.New("generate digest client nonce")
	}
	cnonce := hex.EncodeToString(cnonceBytes)
	nonceCount := "00000001"
	ha1 := digestHash(algorithm, username+":"+realm+":"+password)
	ha2 := digestHash(algorithm, method+":"+requestURI)
	response := digestHash(algorithm, strings.Join([]string{
		ha1, nonce, nonceCount, cnonce, "auth", ha2,
	}, ":"))

	parts := []string{
		`username="` + escapeDigestQuoted(username) + `"`,
		`realm="` + escapeDigestQuoted(realm) + `"`,
		`nonce="` + escapeDigestQuoted(nonce) + `"`,
		`uri="` + escapeDigestQuoted(requestURI) + `"`,
		`response="` + response + `"`,
		"algorithm=" + algorithm,
		"qop=auth",
		"nc=" + nonceCount,
		`cnonce="` + cnonce + `"`,
	}
	if opaque := parameters["opaque"]; opaque != "" {
		parts = append(parts, `opaque="`+escapeDigestQuoted(opaque)+`"`)
	}
	return "Digest " + strings.Join(parts, ", "), nil
}

func parseDigestChallenge(challenge string) (map[string]string, error) {
	challenge = strings.TrimSpace(challenge)
	if len(challenge) < len("Digest ") || !strings.EqualFold(challenge[:len("Digest")], "Digest") {
		return nil, errors.New("authentication challenge is not Digest")
	}
	input := strings.TrimSpace(challenge[len("Digest"):])
	parameters := make(map[string]string)
	for len(input) > 0 {
		input = strings.TrimLeft(input, " ,\t")
		if input == "" {
			break
		}
		equals := strings.IndexByte(input, '=')
		if equals <= 0 {
			return nil, errors.New("digest challenge parameter is invalid")
		}
		name := strings.ToLower(strings.TrimSpace(input[:equals]))
		input = strings.TrimSpace(input[equals+1:])
		var value string
		if strings.HasPrefix(input, `"`) {
			input = input[1:]
			var builder strings.Builder
			escaped := false
			closed := false
			for index, character := range input {
				if escaped {
					builder.WriteRune(character)
					escaped = false
					continue
				}
				if character == '\\' {
					escaped = true
					continue
				}
				if character == '"' {
					value = builder.String()
					input = input[index+1:]
					closed = true
					break
				}
				builder.WriteRune(character)
			}
			if !closed {
				return nil, errors.New("digest challenge quoted value is invalid")
			}
		} else {
			comma := strings.IndexByte(input, ',')
			if comma < 0 {
				value = strings.TrimSpace(input)
				input = ""
			} else {
				value = strings.TrimSpace(input[:comma])
				input = input[comma+1:]
			}
		}
		if name == "" {
			return nil, errors.New("digest challenge parameter name is invalid")
		}
		parameters[name] = value
	}
	return parameters, nil
}

func digestQOPSupportsAuth(value string) bool {
	for _, candidate := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), "auth") {
			return true
		}
	}
	return false
}

func digestHash(algorithm, value string) string {
	if strings.EqualFold(algorithm, "SHA-256") {
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	}
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func escapeDigestQuoted(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func writeConnectionRequestLog(res ConnectionRequestResult, deviceID uint, _ int) {
	status := "OK"
	if res.Err != nil {
		status = "ERR"
	}
	errorCategory := connectionRequestErrorCategory(res.Err)
	s := fmt.Sprintf("----- TR069 CONNECTION REQUEST BEGIN -----\nstatus: %s\ndeviceId: %d\nhttpStatus: %d\nelapsed: %s\nerrorCategory: %s\n----- TR069 CONNECTION REQUEST END -----",
		status, deviceID, res.StatusCode, res.Elapsed.String(), errorCategory)
	_, _ = fmt.Fprintln(os.Stdout, s)
	infolog.Write(s)
	if res.Err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Warn("TR069 connection request failed",
			zap.Uint("deviceId", deviceID),
			zap.String("status", status),
			zap.Int("httpStatus", res.StatusCode),
			zap.Duration("elapsed", res.Elapsed),
			zap.String("errorCategory", errorCategory),
		)
	}
}
