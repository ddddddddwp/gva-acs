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
}
