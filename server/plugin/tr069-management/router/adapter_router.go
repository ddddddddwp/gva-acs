package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/api"
	"github.com/gin-gonic/gin"
)

type AdapterRouter struct{}

func (r *AdapterRouter) InitAdapterRouter(Router *gin.RouterGroup) {
	adapterRouter := Router.Group("adapter").Use(middleware.OperationRecord())
	adapterApi := api.ApiGroupApp.AdapterApi
	{
		adapterRouter.POST("deviceStatus", adapterApi.GetDeviceStatus)       // 获取设备在线状态
		adapterRouter.POST("triggerAction", adapterApi.TriggerDeviceAction)  // 触发设备操作
		adapterRouter.POST("deviceEvents", adapterApi.GetDeviceEvents)       // 获取设备事件记录
		adapterRouter.POST("deviceSessions", adapterApi.GetDeviceSessions)   // 获取设备会话记录
		adapterRouter.POST("deviceLogs", adapterApi.GetDeviceOperationLogs)  // 获取设备操作日志
	}
}