package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/router"
	"github.com/gin-gonic/gin"
)

// InitializeRouter 初始化路由
func InitializeRouter(Router *gin.RouterGroup) {
	{
		router.RouterGroupApp.DeviceRouter.InitDeviceRouter(Router)
		router.RouterGroupApp.ParameterRouter.InitParameterRouter(Router)
		router.RouterGroupApp.AdapterRouter.InitAdapterRouter(Router)
	}
}