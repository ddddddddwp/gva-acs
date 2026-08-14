package adapter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

type EncryptedCredential struct {
	Version    string
	Ciphertext []byte
}

type CredentialCipher interface {
	Encrypt(plaintext string) (EncryptedCredential, error)
	Decrypt(version string, ciphertext []byte) (string, error)
}

type runtimeCredentialCipher struct{}

// NewRuntimeCredentialCipher resolves keys from the latest immutable runtime
// snapshot for every operation so configuration hot reload and key rotation
// apply without rebuilding API or engine singletons.
func NewRuntimeCredentialCipher() CredentialCipher {
	return runtimeCredentialCipher{}
}

func (runtimeCredentialCipher) Encrypt(plaintext string) (EncryptedCredential, error) {
	cipher, err := NewCredentialCipher(config.CurrentRuntime().Settings.ConnectionRequest)
	if err != nil {
		return EncryptedCredential{}, err
	}
	return cipher.Encrypt(plaintext)
}

func (runtimeCredentialCipher) Decrypt(version string, ciphertext []byte) (string, error) {
	cipher, err := NewCredentialCipher(config.CurrentRuntime().Settings.ConnectionRequest)
	if err != nil {
		return "", err
	}
	return cipher.Decrypt(version, ciphertext)
}

type aesGCMCredentialCipher struct {
	activeVersion string
	keys          map[string][]byte
}

func NewCredentialCipher(settings config.ConnectionRequestConfig) (CredentialCipher, error) {
	activeVersion := strings.TrimSpace(settings.CredentialKeyVersion)
	if activeVersion == "" {
		return nil, errors.New("connection request credential key version is required")
	}
	activeKey, err := decodeCredentialKey(settings.CredentialEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("connection request active credential key is invalid: %w", err)
	}
	keys := map[string][]byte{activeVersion: activeKey}
	for version, encoded := range settings.CredentialDecryptionKeys {
		version = strings.TrimSpace(version)
		if version == "" || version == activeVersion {
			continue
		}
		key, err := decodeCredentialKey(encoded)
		if err != nil {
			return nil, fmt.Errorf("connection request historical credential key %q is invalid: %w", version, err)
		}
		keys[version] = key
	}
	return &aesGCMCredentialCipher{activeVersion: activeVersion, keys: keys}, nil
}

func decodeCredentialKey(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, errors.New("key material is required")
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("key material must be base64")
	}
	if len(key) != 32 {
		return nil, errors.New("key material must decode to 32 bytes")
	}
	return append([]byte(nil), key...), nil
}

func (c *aesGCMCredentialCipher) Encrypt(plaintext string) (EncryptedCredential, error) {
	if c == nil {
		return EncryptedCredential{}, errors.New("credential cipher is required")
	}
	key, ok := c.keys[c.activeVersion]
	if !ok {
		return EncryptedCredential{}, errors.New("active credential key is unavailable")
	}
	gcm, err := newCredentialGCM(key)
	if err != nil {
		return EncryptedCredential{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedCredential{}, errors.New("generate credential nonce")
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), []byte(c.activeVersion))
	payload := append(nonce, sealed...)
	return EncryptedCredential{Version: c.activeVersion, Ciphertext: payload}, nil
}

func (c *aesGCMCredentialCipher) Decrypt(version string, ciphertext []byte) (string, error) {
	if c == nil {
		return "", errors.New("credential cipher is required")
	}
	key, ok := c.keys[version]
	if !ok {
		return "", fmt.Errorf("credential key version %q is unavailable", version)
	}
	gcm, err := newCredentialGCM(key)
	if err != nil {
		return "", err
	}
	if len(ciphertext) <= gcm.NonceSize() {
		return "", errors.New("encrypted credential is malformed")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	sealed := ciphertext[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, []byte(version))
	if err != nil {
		return "", errors.New("encrypted credential authentication failed")
	}
	return string(plain), nil
}

func newCredentialGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("initialize credential cipher")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("initialize credential AEAD")
	}
	return gcm, nil
}
