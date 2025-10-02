// Package auth provides authentication utilities for TR069 communication.
// 包 auth 提供了 TR069 通信的认证工具。
package auth

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	httputil "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/http"
)

// AuthManager manages authentication for TR069 communication.
// AuthManager 管理 TR069 通信的认证。
type AuthManager struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
	defaultAuth *Credential
}

// Credential represents authentication credentials.
// Credential 表示认证凭据。
type Credential struct {
	Username string
	Password string
	Realm    string
	Nonce    string
	LastUsed time.Time
}

// NewAuthManager creates a new authentication manager.
// NewAuthManager 创建一个新的认证管理器。
func NewAuthManager() *AuthManager {
	return &AuthManager{
		credentials: make(map[string]*Credential),
	}
}

// SetDefaultCredential sets the default authentication credential.
// SetDefaultCredential 设置默认认证凭据。
func (am *AuthManager) SetDefaultCredential(username, password string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.defaultAuth = &Credential{
		Username: username,
		Password: password,
		LastUsed: time.Now(),
	}
}

// AddCredential adds a credential for a specific realm.
// AddCredential 为特定域添加凭据。
func (am *AuthManager) AddCredential(realm, username, password string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.credentials[realm] = &Credential{
		Username: username,
		Password: password,
		Realm:    realm,
		LastUsed: time.Now(),
	}
}

// GetCredential gets the credential for a specific realm.
// GetCredential 获取特定域的凭据。
func (am *AuthManager) GetCredential(realm string) *Credential {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	if cred, exists := am.credentials[realm]; exists {
		cred.LastUsed = time.Now()
		return cred
	}
	
	// Return default credential if realm-specific not found
	if am.defaultAuth != nil {
		am.defaultAuth.LastUsed = time.Now()
		return am.defaultAuth
	}
	
	return nil
}

// AuthenticateRequest adds authentication to an HTTP request.
// AuthenticateRequest 为 HTTP 请求添加认证。
func (am *AuthManager) AuthenticateRequest(req *http.Request, realm, nonce string) error {
	cred := am.GetCredential(realm)
	if cred == nil {
		return fmt.Errorf("no credential found for realm: %s", realm)
	}
	
	httputil.AddDigestAuth(req, cred.Username, cred.Password, realm, nonce)
	return nil
}

// ValidateDigestAuth validates digest authentication from request.
// ValidateDigestAuth 验证来自请求的 digest 认证。
func (am *AuthManager) ValidateDigestAuth(authHeader, method string) (bool, error) {
	digestAuth, err := httputil.ParseDigestAuth(authHeader)
	if err != nil {
		return false, fmt.Errorf("failed to parse digest auth: %w", err)
	}
	
	cred := am.GetCredential(digestAuth.Realm)
	if cred == nil {
		return false, fmt.Errorf("no credential found for realm: %s", digestAuth.Realm)
	}
	
	if digestAuth.Username != cred.Username {
		return false, fmt.Errorf("username mismatch")
	}
	
	return httputil.ValidateDigestAuth(digestAuth, cred.Password, method), nil
}

// CreateChallenge creates a digest authentication challenge.
// CreateChallenge 创建 digest 认证质询。
func (am *AuthManager) CreateChallenge(realm string) string {
	nonce := httputil.GenerateNonce()
	
	// Update nonce for the realm
	am.mu.Lock()
	if cred, exists := am.credentials[realm]; exists {
		cred.Nonce = nonce
	}
	am.mu.Unlock()
	
	return httputil.CreateDigestChallenge(realm, nonce)
}

// CleanupExpiredCredentials removes expired credentials.
// CleanupExpiredCredentials 清理过期的凭据。
func (am *AuthManager) CleanupExpiredCredentials(expiry time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	now := time.Now()
	for realm, cred := range am.credentials {
		if now.Sub(cred.LastUsed) > expiry {
			delete(am.credentials, realm)
		}
	}
}