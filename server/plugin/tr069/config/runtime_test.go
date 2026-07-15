package config

import (
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
}

func TestStoreRuntimePublishesImmutableSnapshotWithDurations(t *testing.T) {
	in := TR069Config{
		CommandQueueWaitTimeout: 7,
		RPCResponseTimeout:      11,
		TransferCompleteTimeout: 13,
		RPCXMLRetentionDays:     17,
	}

	stored := StoreRuntime(in)
	in.RPCResponseTimeout = 99
	copyOfCurrent := CurrentRuntime()
	copyOfCurrent.Settings.RPCResponseTimeout = 101

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
