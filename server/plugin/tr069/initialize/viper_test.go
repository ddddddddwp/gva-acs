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
