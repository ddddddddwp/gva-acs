package config

type TR069Config struct {
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	Debug   bool   `mapstructure:"debug" json:"debug" yaml:"debug"`
}
