package global

import (
	"sync"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

func TestSetStartupConfigStoresOnlyTheFirstCopiedSnapshot(t *testing.T) {
	previous := GlobalConfig
	startupConfigOnce = sync.Once{}
	t.Cleanup(func() {
		GlobalConfig = previous
		startupConfigOnce = sync.Once{}
	})

	first := config.TR069Config{RPCResponseTimeout: 11}
	SetStartupConfig(first)
	first.RPCResponseTimeout = 22
	SetStartupConfig(config.TR069Config{RPCResponseTimeout: 33})

	if GlobalConfig.RPCResponseTimeout != 11 {
		t.Fatalf("startup response timeout=%d", GlobalConfig.RPCResponseTimeout)
	}
}
