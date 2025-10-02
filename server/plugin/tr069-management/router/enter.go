package router

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/api"

type RouterGroup struct {
	DeviceRouter
	ParameterRouter
	AdapterRouter
}

var RouterGroupApp = new(RouterGroup)
var apiGroupApp = api.ApiGroupApp