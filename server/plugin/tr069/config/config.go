package config

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
}
