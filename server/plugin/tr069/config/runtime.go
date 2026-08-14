package config

import (
	"strings"
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
	defaultRebootConfirmTimeout             = 300
	defaultTransferCompleteTimeout          = 43200
	defaultRPCXMLRetentionDays              = 30
	defaultConnectionRequestTimeout         = 10
	defaultConnectionRequestAuthScheme      = "digest"
	defaultIdentityBindingTTL               = 1800
	defaultFileIngressNonceTTL              = 300
	defaultFileIngressMaxSize               = int64(64 << 20)
	defaultFileIngressConcurrency           = 4
	defaultFileIngressPerDeviceConcurrency  = 1
	defaultFileIngressUploadTimeout         = 600
	defaultFileIngressRetentionDays         = 30
)

type TransferChannelRuntime struct {
	TransferChannelConfig
	UploadTimeout time.Duration
}

type FileIngressRuntime struct {
	IdentityBindingTTL time.Duration
	NonceTTL           time.Duration
	Channels           map[string]TransferChannelRuntime
}

// Runtime is an immutable snapshot of the TR-069 configuration used by workers.
type Runtime struct {
	Settings                 TR069Config
	CommandQueueWaitTimeout  time.Duration
	RPCResponseTimeout       time.Duration
	RebootConfirmTimeout     time.Duration
	TransferCompleteTimeout  time.Duration
	RPCXMLRetention          time.Duration
	ConnectionRequestTimeout time.Duration
	FileIngress              FileIngressRuntime
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
	if in.RebootConfirmTimeout <= 0 {
		in.RebootConfirmTimeout = defaultRebootConfirmTimeout
	}
	if in.TransferCompleteTimeout <= 0 {
		in.TransferCompleteTimeout = defaultTransferCompleteTimeout
	}
	if in.RPCXMLRetentionDays <= 0 {
		in.RPCXMLRetentionDays = defaultRPCXMLRetentionDays
	}
	if in.ConnectionRequest.RequestTimeout <= 0 {
		in.ConnectionRequest.RequestTimeout = defaultConnectionRequestTimeout
	}
	if in.ConnectionRequest.AuthScheme == "" {
		in.ConnectionRequest.AuthScheme = defaultConnectionRequestAuthScheme
	}
	if in.FileIngress.IdentityBindingTTL <= 0 {
		in.FileIngress.IdentityBindingTTL = defaultIdentityBindingTTL
	}
	if len(in.FileIngress.Authentication.Schemes) == 0 {
		in.FileIngress.Authentication.Schemes = []string{"digest", "basic"}
	}
	if in.FileIngress.Authentication.Realm == "" {
		in.FileIngress.Authentication.Realm = "GVA-TR069-LOG"
	}
	if in.FileIngress.Authentication.NonceTTL <= 0 {
		in.FileIngress.Authentication.NonceTTL = defaultFileIngressNonceTTL
	}
	for name, channel := range in.FileIngress.Channels {
		if channel.MaxFileSize <= 0 {
			channel.MaxFileSize = defaultFileIngressMaxSize
		}
		if channel.MaxConcurrent <= 0 {
			channel.MaxConcurrent = defaultFileIngressConcurrency
		}
		if channel.MaxConcurrentPerDevice <= 0 {
			channel.MaxConcurrentPerDevice = defaultFileIngressPerDeviceConcurrency
		}
		if channel.UploadTimeout <= 0 {
			channel.UploadTimeout = defaultFileIngressUploadTimeout
		}
		if channel.RetentionDays <= 0 {
			channel.RetentionDays = defaultFileIngressRetentionDays
		}
		if channel.StoragePrefix == "" {
			channel.StoragePrefix = strings.ToLower(name)
		}
		in.FileIngress.Channels[name] = channel
	}
	if in.FileIngress.ArtifactStore.Prefix == "" {
		in.FileIngress.ArtifactStore.Prefix = "artifacts"
	}
	return in
}

// StoreRuntime publishes a fully constructed immutable runtime snapshot.
func StoreRuntime(in TR069Config) Runtime {
	next := buildRuntime(in)
	current.Store(&next)
	return cloneRuntime(next)
}

func buildRuntime(in TR069Config) Runtime {
	normalized := cloneTR069Config(NormalizeRuntimeConfig(in))
	fileIngress := FileIngressRuntime{
		IdentityBindingTTL: time.Duration(normalized.FileIngress.IdentityBindingTTL) * time.Second,
		NonceTTL:           time.Duration(normalized.FileIngress.Authentication.NonceTTL) * time.Second,
		Channels:           make(map[string]TransferChannelRuntime, len(normalized.FileIngress.Channels)),
	}
	for name, channel := range normalized.FileIngress.Channels {
		fileIngress.Channels[name] = TransferChannelRuntime{TransferChannelConfig: channel, UploadTimeout: time.Duration(channel.UploadTimeout) * time.Second}
	}
	return Runtime{
		Settings:                 normalized,
		CommandQueueWaitTimeout:  time.Duration(normalized.CommandQueueWaitTimeout) * time.Second,
		RPCResponseTimeout:       time.Duration(normalized.RPCResponseTimeout) * time.Second,
		RebootConfirmTimeout:     time.Duration(normalized.RebootConfirmTimeout) * time.Second,
		TransferCompleteTimeout:  time.Duration(normalized.TransferCompleteTimeout) * time.Second,
		RPCXMLRetention:          time.Duration(normalized.RPCXMLRetentionDays) * 24 * time.Hour,
		ConnectionRequestTimeout: time.Duration(normalized.ConnectionRequest.RequestTimeout) * time.Second,
		FileIngress:              fileIngress,
	}
}

func cloneTR069Config(in TR069Config) TR069Config {
	out := in
	out.ConnectionRequest.AllowedCIDRs = append([]string(nil), in.ConnectionRequest.AllowedCIDRs...)
	out.FileIngress.TrustedProxies = append([]string(nil), in.FileIngress.TrustedProxies...)
	out.FileIngress.Authentication.Schemes = append([]string(nil), in.FileIngress.Authentication.Schemes...)
	if in.FileIngress.Channels != nil {
		out.FileIngress.Channels = make(map[string]TransferChannelConfig, len(in.FileIngress.Channels))
		for name, channel := range in.FileIngress.Channels {
			out.FileIngress.Channels[name] = channel
		}
	}
	if in.ConnectionRequest.CredentialDecryptionKeys != nil {
		out.ConnectionRequest.CredentialDecryptionKeys = make(map[string]string, len(in.ConnectionRequest.CredentialDecryptionKeys))
		for version, key := range in.ConnectionRequest.CredentialDecryptionKeys {
			out.ConnectionRequest.CredentialDecryptionKeys[version] = key
		}
	}
	return out
}

func cloneRuntime(in Runtime) Runtime {
	out := in
	out.Settings = cloneTR069Config(in.Settings)
	if in.FileIngress.Channels != nil {
		out.FileIngress.Channels = make(map[string]TransferChannelRuntime, len(in.FileIngress.Channels))
		for name, channel := range in.FileIngress.Channels {
			out.FileIngress.Channels[name] = channel
		}
	}
	return out
}

// CurrentRuntime returns the latest runtime snapshot, initializing defaults on first use.
func CurrentRuntime() Runtime {
	if value := current.Load(); value != nil {
		return cloneRuntime(*value)
	}

	initial := buildRuntime(TR069Config{})
	if current.CompareAndSwap(nil, &initial) {
		return cloneRuntime(initial)
	}
	return cloneRuntime(*current.Load())
}
