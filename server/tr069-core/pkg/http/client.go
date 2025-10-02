// Package http provides HTTP client utilities for TR069 communication.
// 包 http 提供了 TR069 通信的 HTTP 客户端工具。
package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents an HTTP client with digest authentication support.
// Client 表示支持 digest 认证的 HTTP 客户端。
type Client struct {
	httpClient *http.Client
	username   string
	password   string
	timeout    time.Duration
}

// NewClient creates a new HTTP client with digest authentication support.
// NewClient 创建一个支持 digest 认证的新 HTTP 客户端。
func NewClient(username, password string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		username: username,
		password: password,
		timeout:  timeout,
	}
}

// Post sends a POST request with digest authentication.
// Post 发送带有 digest 认证的 POST 请求。
func (c *Client) Post(url string, contentType string, body []byte) (*http.Response, error) {
	// First request without authentication to get the challenge
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send initial request: %w", err)
	}
	
	// If we get 401, try with digest authentication
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		
		// Parse the WWW-Authenticate header
		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			return nil, fmt.Errorf("no WWW-Authenticate header in 401 response")
		}
		
		// Extract digest parameters
		digestParams, err := c.parseWWWAuthenticate(authHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
		}
		
		// Create new request with digest authentication
		req, err = http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)
		
		// Add digest authentication
		AddDigestAuth(req, c.username, c.password, digestParams["realm"], digestParams["nonce"])
		
		// Send authenticated request
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send authenticated request: %w", err)
		}
	}
	
	return resp, nil
}

// Get sends a GET request with digest authentication.
// Get 发送带有 digest 认证的 GET 请求。
func (c *Client) Get(url string) (*http.Response, error) {
	// First request without authentication to get the challenge
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send initial request: %w", err)
	}
	
	// If we get 401, try with digest authentication
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		
		// Parse the WWW-Authenticate header
		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			return nil, fmt.Errorf("no WWW-Authenticate header in 401 response")
		}
		
		// Extract digest parameters
		digestParams, err := c.parseWWWAuthenticate(authHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
		}
		
		// Create new request with digest authentication
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated request: %w", err)
		}
		
		// Add digest authentication
		AddDigestAuth(req, c.username, c.password, digestParams["realm"], digestParams["nonce"])
		
		// Send authenticated request
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send authenticated request: %w", err)
		}
	}
	
	return resp, nil
}

// parseWWWAuthenticate parses the WWW-Authenticate header to extract digest parameters.
// parseWWWAuthenticate 解析 WWW-Authenticate 头部以提取 digest 参数。
func (c *Client) parseWWWAuthenticate(authHeader string) (map[string]string, error) {
	params := make(map[string]string)
	
	// Simple parsing for digest parameters
	// This is a basic implementation and might need enhancement for complex cases
	if !bytes.HasPrefix([]byte(authHeader), []byte("Digest ")) {
		return nil, fmt.Errorf("not a digest authentication header")
	}
	
	authData := authHeader[7:] // Remove "Digest "
	pairs := strings.Split(authData, ",")
	
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		params[key] = value
	}
	
	return params, nil
}

// SetTimeout sets the timeout for HTTP requests.
// SetTimeout 设置 HTTP 请求的超时时间。
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// ReadResponseBody reads and returns the response body as bytes.
// ReadResponseBody 读取并返回响应体的字节数据。
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}