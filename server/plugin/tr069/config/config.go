package config

type TR069Config struct {
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	Debug   bool   `mapstructure:"debug" json:"debug" yaml:"debug"`

	DumpRaw        bool `mapstructure:"dumpRaw" json:"dumpRaw" yaml:"dumpRaw"`
	DumpMaxBytes   int  `mapstructure:"dumpMaxBytes" json:"dumpMaxBytes" yaml:"dumpMaxBytes"`
	DumpRedactAuth bool `mapstructure:"dumpRedactAuth" json:"dumpRedactAuth" yaml:"dumpRedactAuth"`
	DumpRedactCookie bool `mapstructure:"dumpRedactCookie" json:"dumpRedactCookie" yaml:"dumpRedactCookie"`
}
