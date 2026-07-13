package plugin

import (
	_ "github.com/ddddddddwp/gva-acs/server/plugin/announcement"
	_ "github.com/ddddddddwp/gva-acs/server/plugin/auto"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069"
	"github.com/ddddddddwp/gva-acs/server/utils/plugin/v2"
)

func init() {
	plugin.Register(tr069.Plugin)
}
