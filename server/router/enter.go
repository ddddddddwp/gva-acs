package router

import (
	"github.com/ddddddddwp/gva-acs/server/router/example"
	"github.com/ddddddddwp/gva-acs/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System  system.RouterGroup
	Example example.RouterGroup
}
