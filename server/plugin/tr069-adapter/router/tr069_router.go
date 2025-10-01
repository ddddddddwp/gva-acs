package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/api"
	"github.com/gin-gonic/gin"
)

type TR069Router struct{}

func (r *TR069Router) InitTR069Router(Router *gin.RouterGroup) {
	tr069Router := Router
	tr069ApiApp := api.ApiGroupApp.TR069Api
	{
		tr069Router.Use(middleware.OperationRecord())
		tr069Router.GET("device/list", tr069ApiApp.GetDeviceList)       // 获取设备列表
		tr069Router.GET("device/:id", tr069ApiApp.GetDeviceByID)        // 根据ID获取设备
		tr069Router.DELETE("device/:id", tr069ApiApp.DeleteDevice)      // 删除设备
		tr069Router.GET("device/:id/events", tr069ApiApp.GetDeviceEvents) // 获取设备事件
		tr069Router.POST("device/param", tr069ApiApp.SetDeviceParameters) // 设置设备参数
		tr069Router.GET("device/stats", tr069ApiApp.GetDeviceStats)      // 获取设备统计信息
	}
	
	// TR069请求处理路由 - 不需要认证
	Router.POST("cpe", tr069ApiApp.HandleTR069Request)
}