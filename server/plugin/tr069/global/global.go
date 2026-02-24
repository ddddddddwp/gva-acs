package global

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

var GlobalConfig *config.TR069Config

func init() {
	GlobalConfig = &config.TR069Config{}
}

var DataModelValueDenyPrefixes = []string{"Device.FaultMgmt."}
