package config

// TR069AdapterConfig TR069适配器插件配置
type TR069AdapterConfig struct {
	Enabled      bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`            // 是否启用TR069适配器
	ServerPort   int    `mapstructure:"server-port" json:"serverPort" yaml:"server-port"` // TR069服务器端口
	LogLevel     string `mapstructure:"log-level" json:"logLevel" yaml:"log-level"`       // 日志级别
	MaxDevices   int    `mapstructure:"max-devices" json:"maxDevices" yaml:"max-devices"` // 最大设备数量
	EnableCache  bool   `mapstructure:"enable-cache" json:"enableCache" yaml:"enable-cache"`  // 是否启用缓存
	CacheTTL     int    `mapstructure:"cache-ttl" json:"cacheTTL" yaml:"cache-ttl"`      // 缓存过期时间(秒)
}

// DefaultConfig 默认配置
var DefaultConfig = TR069AdapterConfig{
	Enabled:     true,
	ServerPort:  7547,
	LogLevel:    "info",
	MaxDevices:  1000,
	EnableCache: true,
	CacheTTL:    3600,
}