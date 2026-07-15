package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/spf13/viper"
)

func TestApplyConfigChangeTriggersHandlerAfterSuccessfulUnmarshal(t *testing.T) {
	previousEvents := utils.GlobalSystemEvents
	previousConfig := global.GVA_CONFIG
	t.Cleanup(func() {
		utils.GlobalSystemEvents = previousEvents
		global.GVA_CONFIG = previousConfig
	})

	events := &utils.SystemEvents{}
	utils.GlobalSystemEvents = events
	called := 0
	events.RegisterConfigChangeHandler(func() {
		called++
	})

	valid, _ := newConfigFileViper(t, "system:\n  addr: 9999\n")
	if err := applyConfigChange(valid); err != nil {
		t.Fatalf("valid config change: %v", err)
	}
	if called != 1 {
		t.Fatalf("handler calls after valid change=%d", called)
	}

	invalid, _ := newConfigFileViper(t, "system: not-a-system-map\n")
	if err := applyConfigChange(invalid); err == nil {
		t.Fatal("invalid config change unexpectedly unmarshaled")
	}
	if called != 1 {
		t.Fatalf("handler calls after invalid change=%d", called)
	}
}

func TestApplyConfigChangeRejectsMalformedFreshFileWithoutPublishingStaleConfig(t *testing.T) {
	previousEvents := utils.GlobalSystemEvents
	previousConfig := global.GVA_CONFIG
	previousRuntime := tr069Config.CurrentRuntime()
	t.Cleanup(func() {
		utils.GlobalSystemEvents = previousEvents
		global.GVA_CONFIG = previousConfig
		tr069Config.StoreRuntime(previousRuntime.Settings)
	})

	v, path := newConfigFileViper(t, "system:\n  addr: 9999\n")
	global.GVA_CONFIG.System.Addr = 4321
	tr069Config.StoreRuntime(tr069Config.TR069Config{RPCResponseTimeout: 41})

	events := &utils.SystemEvents{}
	utils.GlobalSystemEvents = events
	called := 0
	events.RegisterConfigChangeHandler(func() {
		called++
		tr069Config.StoreRuntime(tr069Config.TR069Config{RPCResponseTimeout: 99})
	})

	if err := os.WriteFile(path, []byte("system: [unterminated\n"), 0o600); err != nil {
		t.Fatalf("replace config with malformed YAML: %v", err)
	}

	if err := applyConfigChange(v); err == nil {
		t.Error("malformed fresh config unexpectedly succeeded")
	}
	if called != 0 {
		t.Errorf("config handler calls=%d, want=0", called)
	}
	if got := global.GVA_CONFIG.System.Addr; got != 4321 {
		t.Errorf("global config addr=%d, want unchanged 4321", got)
	}
	if got := tr069Config.CurrentRuntime().Settings.RPCResponseTimeout; got != 41 {
		t.Errorf("runtime response timeout=%d, want unchanged 41", got)
	}
}

func newConfigFileViper(t *testing.T, contents string) (*viper.Viper, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("initial read config file: %v", err)
	}
	return v, path
}
