package config

import (
	"sync"
	"testing"
	"time"
)

func TestNormalizeRuntimeConfigDefaults(t *testing.T) {
	got := NormalizeRuntimeConfig(TR069Config{})
	if got.CommandQueueWaitTimeout != 180 {
		t.Fatalf("wait=%d", got.CommandQueueWaitTimeout)
	}
	if got.RPCResponseTimeout != 90 {
		t.Fatalf("response=%d", got.RPCResponseTimeout)
	}
	if got.TransferCompleteTimeout != 43200 {
		t.Fatalf("transfer=%d", got.TransferCompleteTimeout)
	}
	if got.RPCXMLRetentionDays != 30 {
		t.Fatalf("retention=%d", got.RPCXMLRetentionDays)
	}
	if got.ConnectionRequest.RequestTimeout != 10 {
		t.Fatalf("connection request timeout=%d", got.ConnectionRequest.RequestTimeout)
	}
	if got.ConnectionRequest.AuthScheme != "digest" {
		t.Fatalf("connection request auth scheme=%q", got.ConnectionRequest.AuthScheme)
	}
}

func TestRebootConfirmTimeoutDefaultsAndPublishesDuration(t *testing.T) {
	normalized := NormalizeRuntimeConfig(TR069Config{})
	if normalized.RebootConfirmTimeout != 300 {
		t.Fatalf("default reboot confirmation timeout = %d", normalized.RebootConfirmTimeout)
	}
	previous := CurrentRuntime()
	t.Cleanup(func() { StoreRuntime(previous.Settings) })
	stored := StoreRuntime(TR069Config{RebootConfirmTimeout: 17})
	if stored.RebootConfirmTimeout != 17*time.Second {
		t.Fatalf("runtime reboot confirmation timeout = %s", stored.RebootConfirmTimeout)
	}
}

func TestStoreRuntimePublishesImmutableSnapshotWithDurations(t *testing.T) {
	in := TR069Config{
		CommandQueueWaitTimeout: 7,
		RPCResponseTimeout:      11,
		TransferCompleteTimeout: 13,
		RPCXMLRetentionDays:     17,
		ConnectionRequest: ConnectionRequestConfig{
			AutoProvisionCredentials: true,
			CredentialKeyVersion:     "v2",
			CredentialEncryptionKey:  "secret-material",
			CredentialDecryptionKeys: map[string]string{"v1": "old-secret-material"},
			RequestTimeout:           19,
			AllowedCIDRs:             []string{"127.0.0.0/8"},
			AuthScheme:               "digest",
		},
	}

	stored := StoreRuntime(in)
	in.RPCResponseTimeout = 99
	in.ConnectionRequest.AllowedCIDRs[0] = "10.0.0.0/8"
	in.ConnectionRequest.CredentialDecryptionKeys["v1"] = "mutated-old-key"
	copyOfCurrent := CurrentRuntime()
	copyOfCurrent.Settings.RPCResponseTimeout = 101
	copyOfCurrent.Settings.ConnectionRequest.AllowedCIDRs[0] = "192.168.0.0/16"
	copyOfCurrent.Settings.ConnectionRequest.CredentialDecryptionKeys["v1"] = "mutated-copy"

	got := CurrentRuntime()
	if got.CommandQueueWaitTimeout != 7*time.Second {
		t.Fatalf("wait duration=%s", got.CommandQueueWaitTimeout)
	}
	if got.RPCResponseTimeout != 11*time.Second {
		t.Fatalf("response duration=%s", got.RPCResponseTimeout)
	}
	if got.TransferCompleteTimeout != 13*time.Second {
		t.Fatalf("transfer duration=%s", got.TransferCompleteTimeout)
	}
	if got.RPCXMLRetention != 17*24*time.Hour {
		t.Fatalf("retention duration=%s", got.RPCXMLRetention)
	}
	if got.ConnectionRequestTimeout != 19*time.Second {
		t.Fatalf("connection request duration=%s", got.ConnectionRequestTimeout)
	}
	if got.Settings.ConnectionRequest.CredentialKeyVersion != "v2" {
		t.Fatalf("connection request key version=%q", got.Settings.ConnectionRequest.CredentialKeyVersion)
	}
	if got.Settings.ConnectionRequest.CredentialEncryptionKey != "secret-material" {
		t.Fatal("connection request encryption key was not preserved in the private runtime snapshot")
	}
	if got.Settings.ConnectionRequest.AllowedCIDRs[0] != "127.0.0.0/8" {
		t.Fatalf("allowed CIDR snapshot mutated to %q", got.Settings.ConnectionRequest.AllowedCIDRs[0])
	}
	if got.Settings.ConnectionRequest.CredentialDecryptionKeys["v1"] != "old-secret-material" {
		t.Fatal("decryption key snapshot was externally mutated")
	}
	if stored.Settings.RPCResponseTimeout != 11 {
		t.Fatalf("stored settings response=%d", stored.Settings.RPCResponseTimeout)
	}
	if got.Settings.RPCResponseTimeout != 11 {
		t.Fatalf("current settings response=%d", got.Settings.RPCResponseTimeout)
	}
}

func TestReloadChangesRuntimeWithoutChangingExistingDeadline(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	initial := StoreRuntime(TR069Config{RPCResponseTimeout: 90})
	deadline := base.Add(initial.RPCResponseTimeout)

	StoreRuntime(TR069Config{RPCResponseTimeout: 5})

	if got := CurrentRuntime().RPCResponseTimeout; got != 5*time.Second {
		t.Fatalf("reloaded response duration=%s", got)
	}
	if want := base.Add(90 * time.Second); !deadline.Equal(want) {
		t.Fatalf("existing deadline=%s want=%s", deadline, want)
	}
}

func TestCurrentRuntimeConcurrentFirstReadDoesNotOverwriteStoredRuntime(t *testing.T) {
	previous := current.Load()
	t.Cleanup(func() {
		current.Store(previous)
	})

	const iterations = 2000
	for iteration := 0; iteration < iterations; iteration++ {
		current.Store(nil)
		start := make(chan struct{})
		var wait sync.WaitGroup
		wait.Add(2)

		go func() {
			defer wait.Done()
			<-start
			_ = CurrentRuntime()
		}()
		go func() {
			defer wait.Done()
			<-start
			StoreRuntime(TR069Config{RPCResponseTimeout: 777})
		}()

		close(start)
		wait.Wait()
		if got := CurrentRuntime().Settings.RPCResponseTimeout; got != 777 {
			t.Fatalf("iteration %d: lazy defaults overwrote stored response timeout: %d", iteration, got)
		}
	}
}
