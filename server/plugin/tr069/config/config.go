package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

type FileIngressAuthConfig struct {
	Username string   `mapstructure:"username" json:"-" yaml:"username"`
	Password string   `mapstructure:"password" json:"-" yaml:"password"`
	Schemes  []string `mapstructure:"schemes" json:"schemes" yaml:"schemes"`
	Realm    string   `mapstructure:"realm" json:"realm" yaml:"realm"`
	NonceTTL int      `mapstructure:"nonceTTL" json:"nonceTTL" yaml:"nonceTTL"`
}

type TransferChannelConfig struct {
	Enabled                bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Path                   string `mapstructure:"path" json:"path" yaml:"path"`
	MaxFileSize            int64  `mapstructure:"maxFileSize" json:"maxFileSize" yaml:"maxFileSize"`
	MaxConcurrent          int    `mapstructure:"maxConcurrent" json:"maxConcurrent" yaml:"maxConcurrent"`
	MaxConcurrentPerDevice int    `mapstructure:"maxConcurrentPerDevice" json:"maxConcurrentPerDevice" yaml:"maxConcurrentPerDevice"`
	UploadTimeout          int    `mapstructure:"uploadTimeout" json:"uploadTimeout" yaml:"uploadTimeout"`
	RetentionDays          int    `mapstructure:"retentionDays" json:"retentionDays" yaml:"retentionDays"`
	StoragePrefix          string `mapstructure:"storagePrefix" json:"storagePrefix" yaml:"storagePrefix"`
}

type ArtifactStoreConfig struct {
	Driver    string `mapstructure:"driver" json:"driver" yaml:"driver"`
	Endpoint  string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Bucket    string `mapstructure:"bucket" json:"bucket" yaml:"bucket"`
	AccessKey string `mapstructure:"accessKey" json:"-" yaml:"accessKey"`
	SecretKey string `mapstructure:"secretKey" json:"-" yaml:"secretKey"`
	UseSSL    bool   `mapstructure:"useSSL" json:"useSSL" yaml:"useSSL"`
	Prefix    string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
}

type FileIngressConfig struct {
	Enabled            bool                             `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	PublicBaseURL      string                           `mapstructure:"publicBaseURL" json:"publicBaseURL" yaml:"publicBaseURL"`
	TrustedProxies     []string                         `mapstructure:"trustedProxies" json:"trustedProxies" yaml:"trustedProxies"`
	IdentityBindingTTL int                              `mapstructure:"identityBindingTTL" json:"identityBindingTTL" yaml:"identityBindingTTL"`
	Authentication     FileIngressAuthConfig            `mapstructure:"authentication" json:"authentication" yaml:"authentication"`
	Channels           map[string]TransferChannelConfig `mapstructure:"channels" json:"channels" yaml:"channels"`
	ArtifactStore      ArtifactStoreConfig              `mapstructure:"artifactStore" json:"artifactStore" yaml:"artifactStore"`
}

type ConnectionRequestConfig struct {
	AutoProvisionCredentials bool              `mapstructure:"autoProvisionCredentials" json:"autoProvisionCredentials" yaml:"autoProvisionCredentials"`
	CredentialKeyVersion     string            `mapstructure:"credentialKeyVersion" json:"credentialKeyVersion" yaml:"credentialKeyVersion"`
	CredentialEncryptionKey  string            `mapstructure:"credentialEncryptionKey" json:"-" yaml:"credentialEncryptionKey"`
	CredentialDecryptionKeys map[string]string `mapstructure:"credentialDecryptionKeys" json:"-" yaml:"credentialDecryptionKeys"`
	RequestTimeout           int               `mapstructure:"requestTimeout" json:"requestTimeout" yaml:"requestTimeout"`
	AllowedCIDRs             []string          `mapstructure:"allowedCIDRs" json:"allowedCIDRs" yaml:"allowedCIDRs"`
	AuthScheme               string            `mapstructure:"authScheme" json:"authScheme" yaml:"authScheme"`
}

type TR069Config struct {
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	Debug   bool   `mapstructure:"debug" json:"debug" yaml:"debug"`

	// DumpRaw: TR069 调试开关，开启后在终端打印 CPE 原始 HTTP 报文（不解析）。
	// 对应配置：config.yaml -> tr069.dumpRaw
	DumpRaw bool `mapstructure:"dumpRaw" json:"dumpRaw" yaml:"dumpRaw"`

	// DumpMaxBytes: 单次请求最多打印字节数（超出截断）。
	DumpMaxBytes int `mapstructure:"dumpMaxBytes" json:"dumpMaxBytes" yaml:"dumpMaxBytes"`

	// DumpRedactAuth: DumpRaw 时是否脱敏 Authorization。
	DumpRedactAuth bool `mapstructure:"dumpRedactAuth" json:"dumpRedactAuth" yaml:"dumpRedactAuth"`

	// DumpRedactCookie: DumpRaw 时是否脱敏 Cookie。
	DumpRedactCookie bool `mapstructure:"dumpRedactCookie" json:"dumpRedactCookie" yaml:"dumpRedactCookie"`
	// InfoLogEnable: 是否将 TR069 原始报文额外写入文件（避免终端日志截断）
	// 对应配置：config.yaml -> tr069.infoLogEnable
	InfoLogEnable bool `mapstructure:"infoLogEnable" json:"infoLogEnable" yaml:"infoLogEnable"`

	// InfoLogDir: TR069 原始报文日志根目录（相对 server 工作目录），默认 "./log"
	// 文件路径格式：<InfoLogDir>/<YYYY-MM-DD>/tr069info.log
	// 对应配置：config.yaml -> tr069.infoLogDir
	InfoLogDir string `mapstructure:"infoLogDir" json:"infoLogDir" yaml:"infoLogDir"`

	// ==================== 队列配置 ====================

	// CommandQueueLockTTL: 设备级别分布式锁的 TTL（秒），默认 30 秒
	CommandQueueLockTTL int `mapstructure:"commandQueueLockTTL" json:"commandQueueLockTTL" yaml:"commandQueueLockTTL"`

	// CommandQueueDedupTTL: 去重 key 的 TTL（秒），默认 86400 秒（24 小时）
	CommandQueueDedupTTL int `mapstructure:"commandQueueDedupTTL" json:"commandQueueDedupTTL" yaml:"commandQueueDedupTTL"`

	// CommandQueueMaxScan: 每次从队列获取命令的最大扫描次数，默认 10
	CommandQueueMaxScan int `mapstructure:"commandQueueMaxScan" json:"commandQueueMaxScan" yaml:"commandQueueMaxScan"`

	// CommandQueueMaxPendingPerSession: 每次 session 最多处理的 pending 命令数量，避免饥饿，默认 5
	CommandQueueMaxPendingPerSession int `mapstructure:"commandQueueMaxPendingPerSession" json:"commandQueueMaxPendingPerSession" yaml:"commandQueueMaxPendingPerSession"`

	// CommandQueueImmediateTTL: 立即下发队列的 TTL（秒），默认 1800 秒（30 分钟）
	CommandQueueImmediateTTL int `mapstructure:"commandQueueImmediateTTL" json:"commandQueueImmediateTTL" yaml:"commandQueueImmediateTTL"`

	// CommandQueueWaitTimeout: 命令等待设备上线的超时时间（秒），默认 180 秒
	CommandQueueWaitTimeout int `mapstructure:"commandQueueWaitTimeout" json:"commandQueueWaitTimeout" yaml:"commandQueueWaitTimeout"`

	// RPCResponseTimeout: RPC 发送后等待响应的超时时间（秒），默认 90 秒
	RPCResponseTimeout int `mapstructure:"rpcResponseTimeout" json:"rpcResponseTimeout" yaml:"rpcResponseTimeout"`

	// RebootConfirmTimeout: Reboot 响应后等待设备启动 Inform 的超时时间（秒），默认 300 秒
	RebootConfirmTimeout int `mapstructure:"rebootConfirmTimeout" json:"rebootConfirmTimeout" yaml:"rebootConfirmTimeout"`

	// TransferCompleteTimeout: 下载或上传后等待 TransferComplete 的超时时间（秒），默认 43200 秒
	TransferCompleteTimeout int `mapstructure:"transferCompleteTimeout" json:"transferCompleteTimeout" yaml:"transferCompleteTimeout"`

	// RPCXMLRetentionDays: RPC XML 的保留天数，默认 30 天
	RPCXMLRetentionDays int `mapstructure:"rpcXMLRetentionDays" json:"rpcXMLRetentionDays" yaml:"rpcXMLRetentionDays"`

	// ConnectionRequest controls CPE wakeup and per-device credential provisioning.
	ConnectionRequest ConnectionRequestConfig `mapstructure:"connectionRequest" json:"connectionRequest" yaml:"connectionRequest"`

	FileIngress FileIngressConfig `mapstructure:"fileIngress" json:"fileIngress" yaml:"fileIngress"`
}

func ValidateFileIngress(in TR069Config) error {
	cfg := in.FileIngress
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Authentication.Username) == "" {
		return errors.New("tr069 file ingress username is required")
	}
	if strings.TrimSpace(cfg.Authentication.Password) == "" {
		return errors.New("tr069 file ingress password is required")
	}
	if strings.TrimSpace(cfg.Authentication.Realm) == "" {
		return errors.New("tr069 file ingress realm is required")
	}
	if cfg.IdentityBindingTTL <= 0 || cfg.Authentication.NonceTTL <= 0 {
		return errors.New("tr069 file ingress TTL values must be positive")
	}
	parsedBase, err := url.Parse(strings.TrimSpace(cfg.PublicBaseURL))
	if err != nil || parsedBase.Host == "" || (parsedBase.Scheme != "http" && parsedBase.Scheme != "https") {
		return errors.New("tr069 file ingress publicBaseURL must be an absolute HTTP URL")
	}
	for _, raw := range cfg.TrustedProxies {
		if _, err := netip.ParsePrefix(strings.TrimSpace(raw)); err != nil {
			return fmt.Errorf("tr069 file ingress trusted proxy CIDR is invalid: %q", raw)
		}
	}
	if len(cfg.Authentication.Schemes) == 0 {
		return errors.New("tr069 file ingress authentication schemes are required")
	}
	for _, scheme := range cfg.Authentication.Schemes {
		switch strings.ToLower(strings.TrimSpace(scheme)) {
		case "basic", "digest":
		default:
			return fmt.Errorf("tr069 file ingress authentication scheme is unsupported: %q", scheme)
		}
	}
	paths := make(map[string]string, len(cfg.Channels))
	enabled := 0
	for name, channel := range cfg.Channels {
		if !channel.Enabled {
			continue
		}
		enabled++
		if !strings.HasPrefix(channel.Path, "/acs/") {
			return fmt.Errorf("tr069 file ingress channel %q path must start with /acs/", name)
		}
		if existing, ok := paths[channel.Path]; ok {
			return fmt.Errorf("tr069 file ingress channels %q and %q use the same path", existing, name)
		}
		paths[channel.Path] = name
		if channel.MaxFileSize <= 0 || channel.MaxConcurrent <= 0 || channel.MaxConcurrentPerDevice <= 0 || channel.UploadTimeout <= 0 || channel.RetentionDays <= 0 {
			return fmt.Errorf("tr069 file ingress channel %q limits must be positive", name)
		}
	}
	if enabled == 0 {
		return errors.New("tr069 file ingress requires at least one enabled channel")
	}
	store := cfg.ArtifactStore
	if strings.ToLower(strings.TrimSpace(store.Driver)) != "minio" {
		return errors.New("tr069 artifact store driver must be minio")
	}
	if strings.TrimSpace(store.Endpoint) == "" || strings.TrimSpace(store.Bucket) == "" || strings.TrimSpace(store.AccessKey) == "" || strings.TrimSpace(store.SecretKey) == "" {
		return errors.New("tr069 minio endpoint, bucket, accessKey, and secretKey are required")
	}
	return nil
}
