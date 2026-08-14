package initialize

import (
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/spf13/viper"
)

func TestReloadConfigPublishesRuntimeWithoutMutatingLegacyConfig(t *testing.T) {
	previousViper := global.GVA_VP
	previousLegacy := tr069Global.GlobalConfig
	previousRuntime := tr069Config.CurrentRuntime()
	t.Cleanup(func() {
		global.GVA_VP = previousViper
		tr069Global.GlobalConfig = previousLegacy
		tr069Config.StoreRuntime(previousRuntime.Settings)
	})

	legacy := &tr069Config.TR069Config{RPCResponseTimeout: 77}
	tr069Global.GlobalConfig = legacy
	global.GVA_VP = viper.New()
	global.GVA_VP.Set("tr069.commandQueueWaitTimeout", 21)
	global.GVA_VP.Set("tr069.rpcResponseTimeout", 34)
	global.GVA_VP.Set("tr069.transferCompleteTimeout", 55)
	global.GVA_VP.Set("tr069.rebootConfirmTimeout", 67)
	global.GVA_VP.Set("tr069.rpcXMLRetentionDays", 8)

	ReloadConfig()
	ReloadConfig()

	got := tr069Config.CurrentRuntime()
	if got.CommandQueueWaitTimeout != 21*time.Second {
		t.Fatalf("wait duration=%s", got.CommandQueueWaitTimeout)
	}
	if got.RPCResponseTimeout != 34*time.Second {
		t.Fatalf("response duration=%s", got.RPCResponseTimeout)
	}
	if got.TransferCompleteTimeout != 55*time.Second {
		t.Fatalf("transfer duration=%s", got.TransferCompleteTimeout)
	}
	if got.RebootConfirmTimeout != 67*time.Second {
		t.Fatalf("reboot confirmation duration=%s", got.RebootConfirmTimeout)
	}
	if got.RPCXMLRetention != 8*24*time.Hour {
		t.Fatalf("retention duration=%s", got.RPCXMLRetention)
	}
	if tr069Global.GlobalConfig != legacy {
		t.Fatal("ReloadConfig replaced the legacy startup config pointer")
	}
	if legacy.RPCResponseTimeout != 77 {
		t.Fatalf("legacy response timeout mutated to %d", legacy.RPCResponseTimeout)
	}
}

func TestReloadConfigDoesNotPublishInvalidFileIngressConfig(t *testing.T) {
	previousViper := global.GVA_VP
	previousRuntime := tr069Config.CurrentRuntime()
	t.Cleanup(func() {
		global.GVA_VP = previousViper
		tr069Config.StoreRuntime(previousRuntime.Settings)
	})

	tr069Config.StoreRuntime(tr069Config.TR069Config{RPCResponseTimeout: 41})
	global.GVA_VP = viper.New()
	global.GVA_VP.Set("tr069.rpcResponseTimeout", 99)
	global.GVA_VP.Set("tr069.fileIngress.enabled", true)
	global.GVA_VP.Set("tr069.fileIngress.publicBaseURL", "http://127.0.0.1:7458")
	global.GVA_VP.Set("tr069.fileIngress.authentication.username", "shared-log-user")
	global.GVA_VP.Set("tr069.fileIngress.authentication.password", "")
	global.GVA_VP.Set("tr069.fileIngress.authentication.realm", "GVA-TR069")
	global.GVA_VP.Set("tr069.fileIngress.channels.log.enabled", true)
	global.GVA_VP.Set("tr069.fileIngress.channels.log.path", "/acs/log")
	global.GVA_VP.Set("tr069.fileIngress.store.driver", "minio")
	global.GVA_VP.Set("tr069.fileIngress.store.endpoint", "127.0.0.1:9000")
	global.GVA_VP.Set("tr069.fileIngress.store.bucket", "tr069-artifacts")
	global.GVA_VP.Set("tr069.fileIngress.store.accessKey", "minio")
	global.GVA_VP.Set("tr069.fileIngress.store.secretKey", "secret")

	ReloadConfig()

	got := tr069Config.CurrentRuntime()
	if got.Settings.RPCResponseTimeout != 41 {
		t.Fatalf("invalid config published response timeout=%d", got.Settings.RPCResponseTimeout)
	}
	if got.Settings.FileIngress.Enabled {
		t.Fatal("invalid file ingress config was published")
	}
}

func TestLoadConfigReturnsInvalidFileIngressError(t *testing.T) {
	previousViper := global.GVA_VP
	t.Cleanup(func() { global.GVA_VP = previousViper })

	global.GVA_VP = viper.New()
	global.GVA_VP.Set("tr069.fileIngress.enabled", true)
	global.GVA_VP.Set("tr069.fileIngress.publicBaseURL", "http://127.0.0.1:7458")
	global.GVA_VP.Set("tr069.fileIngress.authentication.username", "shared-log-user")
	global.GVA_VP.Set("tr069.fileIngress.authentication.password", "")

	if err := LoadConfig(); err == nil {
		t.Fatal("invalid enabled file ingress config was accepted")
	}
}
