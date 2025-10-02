package initialize

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/router"
)

// InitializeRouter 初始化路由
func InitializeRouter(Router *gin.RouterGroup) {
	// 直接使用传入的路由组，不再创建子组
	// 因为plugin.go中已经创建了tr069-adapter路由组
	
	// 初始化各个模块的路由
	router.RouterGroupApp.DeviceRouter.InitDeviceRouter(Router)
	router.RouterGroupApp.ParameterRouter.InitParameterRouter(Router)
	router.RouterGroupApp.SessionRouter.InitSessionRouter(Router)
	router.RouterGroupApp.OperationLogRouter.InitOperationLogRouter(Router)
	router.RouterGroupApp.TR069Router.InitTR069Router(Router)
}