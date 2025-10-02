// Package http provides HTTP utilities for TR069 communication.
// 包 http 提供了 TR069 通信的 HTTP 工具。
package http

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DigestAuth represents HTTP Digest authentication parameters.
// DigestAuth 表示 HTTP Digest 认证参数。
type DigestAuth struct {
	Username  string
	Password  string
	Realm     string
	Nonce     string
	URI       string
	QOP       string
	NC        string
	CNonce    string
	Response  string
	Algorithm string
}

// ParseDigestAuth parses the Authorization header for Digest authentication.
// ParseDigestAuth 解析 Authorization 头部的 Digest 认证信息。
func ParseDigestAuth(authHeader string) (*DigestAuth, error) {
	if !strings.HasPrefix(authHeader, "Digest ") {
		return nil, fmt.Errorf("not a digest auth header")
	}
	
	authData := strings.TrimPrefix(authHeader, "Digest ")
	auth := &DigestAuth{}
	
	// Parse key-value pairs
	re := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := re.FindAllStringSubmatch(authData, -1)
	
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		key, value := match[1], match[2]
		
		switch key {
		case "username":
			auth.Username = value
		case "realm":
			auth.Realm = value
		case "nonce":
			auth.Nonce = value
		case "uri":
			auth.URI = value
		case "qop":
			auth.QOP = value
		case "nc":
			auth.NC = value
		case "cnonce":
			auth.CNonce = value
		case "response":
			auth.Response = value
		case "algorithm":
			auth.Algorithm = value
		}
	}
	
	return auth, nil
}

// GenerateDigestResponse generates the digest response hash.
// GenerateDigestResponse 生成 digest 响应哈希。
func GenerateDigestResponse(username, password, realm, method, uri, nonce, qop, nc, cnonce string) string {
	// HA1 = MD5(username:realm:password)
	ha1 := fmt.Sprintf("%x", md5.Sum([]byte(username+":"+realm+":"+password)))
	
	// HA2 = MD5(method:uri)
	ha2 := fmt.Sprintf("%x", md5.Sum([]byte(method+":"+uri)))
	
	// Response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
	var response string
	if qop == "auth" || qop == "auth-int" {
		response = fmt.Sprintf("%x", md5.Sum([]byte(ha1+":"+nonce+":"+nc+":"+cnonce+":"+qop+":"+ha2)))
	} else {
		response = fmt.Sprintf("%x", md5.Sum([]byte(ha1+":"+nonce+":"+ha2)))
	}
	
	return response
}

// ValidateDigestAuth validates the digest authentication.
// ValidateDigestAuth 验证 digest 认证。
func ValidateDigestAuth(auth *DigestAuth, password, method string) bool {
	expectedResponse := GenerateDigestResponse(
		auth.Username, password, auth.Realm, method, auth.URI,
		auth.Nonce, auth.QOP, auth.NC, auth.CNonce,
	)
	
	return auth.Response == expectedResponse
}

// GenerateNonce generates a random nonce for digest authentication.
// GenerateNonce 为 digest 认证生成随机 nonce。
func GenerateNonce() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(strconv.FormatInt(time.Now().UnixNano(), 10))))
}

// CreateDigestChallenge creates a WWW-Authenticate header for digest authentication.
// CreateDigestChallenge 创建用于 digest 认证的 WWW-Authenticate 头部。
func CreateDigestChallenge(realm, nonce string) string {
	return fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth", algorithm="MD5"`, realm, nonce)
}

// AddDigestAuth adds digest authentication to an HTTP request.
// AddDigestAuth 为 HTTP 请求添加 digest 认证。
func AddDigestAuth(req *http.Request, username, password, realm, nonce string) {
	uri := req.URL.RequestURI()
	method := req.Method
	qop := "auth"
	nc := "00000001"
	cnonce := GenerateNonce()
	
	response := GenerateDigestResponse(username, password, realm, method, uri, nonce, qop, nc, cnonce)
	
	authHeader := fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop="%s", nc=%s, cnonce="%s", response="%s", algorithm="MD5"`,
		username, realm, nonce, uri, qop, nc, cnonce, response,
	)
	
	req.Header.Set("Authorization", authHeader)
}