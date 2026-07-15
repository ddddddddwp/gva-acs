package global

import (
	"sync"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

var (
	GlobalConfig      = &config.TR069Config{}
	startupConfigOnce sync.Once
)

// SetStartupConfig stores the first configuration as a legacy compatibility
// snapshot. Reloads publish through config.StoreRuntime and never mutate it.
func SetStartupConfig(settings config.TR069Config) {
	startupConfigOnce.Do(func() {
		snapshot := settings
		GlobalConfig = &snapshot
	})
}

var DataModelValueDenyPrefixes = []string{"Device.FaultMgmt."}
