package config

import (
	"sync/atomic"
	"time"
)

const (
	defaultCommandQueueLockTTL              = 30
	defaultCommandQueueDedupTTL             = 86400
	defaultCommandQueueMaxScan              = 10
	defaultCommandQueueMaxPendingPerSession = 5
	defaultCommandQueueImmediateTTL         = 1800
	defaultCommandQueueWaitTimeout          = 180
	defaultRPCResponseTimeout               = 90
	defaultTransferCompleteTimeout          = 43200
	defaultRPCXMLRetentionDays              = 30
)

// Runtime is an immutable snapshot of the TR-069 configuration used by workers.
type Runtime struct {
	Settings                TR069Config
	CommandQueueWaitTimeout time.Duration
	RPCResponseTimeout      time.Duration
	TransferCompleteTimeout time.Duration
	RPCXMLRetention         time.Duration
}

var current atomic.Pointer[Runtime]

// NormalizeRuntimeConfig applies safe defaults without mutating the input.
func NormalizeRuntimeConfig(in TR069Config) TR069Config {
	if in.InfoLogDir == "" {
		in.InfoLogDir = "./log"
	}
	if in.CommandQueueLockTTL <= 0 {
		in.CommandQueueLockTTL = defaultCommandQueueLockTTL
	}
	if in.CommandQueueDedupTTL <= 0 {
		in.CommandQueueDedupTTL = defaultCommandQueueDedupTTL
	}
	if in.CommandQueueMaxScan <= 0 {
		in.CommandQueueMaxScan = defaultCommandQueueMaxScan
	}
	if in.CommandQueueMaxPendingPerSession <= 0 {
		in.CommandQueueMaxPendingPerSession = defaultCommandQueueMaxPendingPerSession
	}
	if in.CommandQueueImmediateTTL <= 0 {
		in.CommandQueueImmediateTTL = defaultCommandQueueImmediateTTL
	}
	if in.CommandQueueWaitTimeout <= 0 {
		in.CommandQueueWaitTimeout = defaultCommandQueueWaitTimeout
	}
	if in.RPCResponseTimeout <= 0 {
		in.RPCResponseTimeout = defaultRPCResponseTimeout
	}
	if in.TransferCompleteTimeout <= 0 {
		in.TransferCompleteTimeout = defaultTransferCompleteTimeout
	}
	if in.RPCXMLRetentionDays <= 0 {
		in.RPCXMLRetentionDays = defaultRPCXMLRetentionDays
	}
	return in
}

// StoreRuntime publishes a fully constructed immutable runtime snapshot.
func StoreRuntime(in TR069Config) Runtime {
	next := buildRuntime(in)
	current.Store(&next)
	return next
}

func buildRuntime(in TR069Config) Runtime {
	normalized := NormalizeRuntimeConfig(in)
	return Runtime{
		Settings:                normalized,
		CommandQueueWaitTimeout: time.Duration(normalized.CommandQueueWaitTimeout) * time.Second,
		RPCResponseTimeout:      time.Duration(normalized.RPCResponseTimeout) * time.Second,
		TransferCompleteTimeout: time.Duration(normalized.TransferCompleteTimeout) * time.Second,
		RPCXMLRetention:         time.Duration(normalized.RPCXMLRetentionDays) * 24 * time.Hour,
	}
}

// CurrentRuntime returns the latest runtime snapshot, initializing defaults on first use.
func CurrentRuntime() Runtime {
	if value := current.Load(); value != nil {
		return *value
	}

	initial := buildRuntime(TR069Config{})
	if current.CompareAndSwap(nil, &initial) {
		return initial
	}
	return *current.Load()
}
