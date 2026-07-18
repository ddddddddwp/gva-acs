package config

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestRepositoryConfigsContainSafeFileIngressDefaults(t *testing.T) {
	for _, filename := range []string{"config.yaml", "config.docker.yaml"} {
		t.Run(filename, func(t *testing.T) {
			reader := viper.New()
			reader.SetConfigFile(filepath.Join("..", "..", "..", filename))
			if err := reader.ReadInConfig(); err != nil {
				t.Fatalf("parse %s: %v", filename, err)
			}
			var cfg TR069Config
			if err := reader.UnmarshalKey("tr069", &cfg); err != nil {
				t.Fatalf("decode tr069 in %s: %v", filename, err)
			}
			if cfg.FileIngress.Enabled {
				t.Fatalf("%s must keep file ingress disabled by default", filename)
			}
			if cfg.FileIngress.Authentication.Username != "" || cfg.FileIngress.Authentication.Password != "" {
				t.Fatalf("%s committed real shared credentials", filename)
			}
			logChannel, ok := cfg.FileIngress.Channels["log"]
			if !ok || !logChannel.Enabled || logChannel.Path != "/acs/log" || logChannel.MaxFileSize != 64<<20 {
				t.Fatalf("%s LOG defaults = %#v", filename, logChannel)
			}
			for _, channel := range []string{"pm", "mr"} {
				if cfg.FileIngress.Channels[channel].Enabled {
					t.Fatalf("%s channel %s must be disabled", filename, channel)
				}
			}
			if cfg.FileIngress.ArtifactStore.Driver != "minio" || cfg.FileIngress.ArtifactStore.Bucket != "gva-tr069-artifacts" {
				t.Fatalf("%s store defaults = %#v", filename, cfg.FileIngress.ArtifactStore)
			}
		})
	}
}
