package adapter

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

func testCredentialConfig(fill byte) config.ConnectionRequestConfig {
	key := bytes.Repeat([]byte{fill}, 32)
	return config.ConnectionRequestConfig{
		CredentialKeyVersion:    "v1",
		CredentialEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}
}

func TestCredentialCipherRoundTripUsesUniqueNonce(t *testing.T) {
	cipher, err := NewCredentialCipher(testCredentialConfig(0x41))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	a, err := cipher.Encrypt("device-secret")
	if err != nil {
		t.Fatalf("encrypt first: %v", err)
	}
	b, err := cipher.Encrypt("device-secret")
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}
	if a.Version != "v1" || b.Version != "v1" {
		t.Fatalf("versions=%q/%q", a.Version, b.Version)
	}
	if bytes.Equal(a.Ciphertext, b.Ciphertext) {
		t.Fatal("encryptions reused a nonce")
	}
	plain, err := cipher.Decrypt(a.Version, a.Ciphertext)
	if err != nil || plain != "device-secret" {
		t.Fatalf("decrypt=(%q,%v)", plain, err)
	}
}

func TestCredentialCipherRejectsWrongKeyWithoutLeakingSecret(t *testing.T) {
	first, err := NewCredentialCipher(testCredentialConfig(0x42))
	if err != nil {
		t.Fatalf("new first cipher: %v", err)
	}
	second, err := NewCredentialCipher(testCredentialConfig(0x43))
	if err != nil {
		t.Fatalf("new second cipher: %v", err)
	}
	encrypted, err := first.Encrypt("must-not-leak")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = second.Decrypt(encrypted.Version, encrypted.Ciphertext)
	if err == nil {
		t.Fatal("decrypt with wrong key succeeded")
	}
	if strings.Contains(err.Error(), "must-not-leak") || strings.Contains(err.Error(), base64.StdEncoding.EncodeToString(encrypted.Ciphertext)) {
		t.Fatalf("error leaked secret material: %v", err)
	}
}

func TestCredentialCipherReadsConfiguredHistoricalKey(t *testing.T) {
	oldConfig := testCredentialConfig(0x44)
	oldCipher, err := NewCredentialCipher(oldConfig)
	if err != nil {
		t.Fatalf("new old cipher: %v", err)
	}
	encrypted, err := oldCipher.Encrypt("rotated-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	newConfig := testCredentialConfig(0x45)
	newConfig.CredentialKeyVersion = "v2"
	newConfig.CredentialDecryptionKeys = map[string]string{
		"v1": oldConfig.CredentialEncryptionKey,
	}
	newCipher, err := NewCredentialCipher(newConfig)
	if err != nil {
		t.Fatalf("new rotated cipher: %v", err)
	}
	plain, err := newCipher.Decrypt(encrypted.Version, encrypted.Ciphertext)
	if err != nil || plain != "rotated-secret" {
		t.Fatalf("decrypt historical=(%q,%v)", plain, err)
	}
}

func TestRuntimeCredentialCipherUsesHotReloadedKeys(t *testing.T) {
	previous := config.CurrentRuntime()
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })

	oldConfig := testCredentialConfig(0x46)
	settings := previous.Settings
	settings.ConnectionRequest = oldConfig
	config.StoreRuntime(settings)
	runtimeCipher := NewRuntimeCredentialCipher()
	encrypted, err := runtimeCipher.Encrypt("hot-reload-secret")
	if err != nil {
		t.Fatalf("encrypt with initial runtime key: %v", err)
	}

	newConfig := testCredentialConfig(0x47)
	newConfig.CredentialKeyVersion = "v2"
	newConfig.CredentialDecryptionKeys = map[string]string{"v1": oldConfig.CredentialEncryptionKey}
	settings.ConnectionRequest = newConfig
	config.StoreRuntime(settings)
	plain, err := runtimeCipher.Decrypt(encrypted.Version, encrypted.Ciphertext)
	if err != nil || plain != "hot-reload-secret" {
		t.Fatalf("decrypt after runtime rotation=(%q,%v)", plain, err)
	}
	rotated, err := runtimeCipher.Encrypt("new-secret")
	if err != nil || rotated.Version != "v2" {
		t.Fatalf("encrypt after runtime rotation=(%q,%v)", rotated.Version, err)
	}
}
