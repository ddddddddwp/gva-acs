package plugin

import (
	_ "github.com/ddddddddwp/gva-acs/server/plugin/announcement"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069"
	"github.com/ddddddddwp/gva-acs/server/utils/plugin/v2"
)

func init() {
	plugin.Register(tr069.Plugin)
}
