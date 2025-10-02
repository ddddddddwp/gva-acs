package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type DeviceRouter struct{}

// InitDeviceRouter 初始化设备路由
func (r *DeviceRouter) InitDeviceRouter(Router *gin.RouterGroup) {
	deviceRouter := Router.Group("device").Use(middleware.OperationRecord())
	deviceRouterWithoutRecord := Router.Group("device")
	
	deviceApi := apiGroupApp.DeviceApi
	{
		deviceRouter.POST("create", deviceApi.CreateDevice)       // 创建设备
		deviceRouter.PUT("update", deviceApi.UpdateDevice)        // 更新设备
		deviceRouter.DELETE("delete/:id", deviceApi.DeleteDevice) // 删除设备
		deviceRouter.POST("batchDelete", deviceApi.BatchDeleteDevices) // 批量删除设备
		deviceRouter.PUT("status", deviceApi.UpdateDeviceStatus)  // 更新设备状态
	}
	{
		deviceRouterWithoutRecord.GET("list", deviceApi.GetDeviceList)       // 获取设备列表
		deviceRouterWithoutRecord.GET("detail/:id", deviceApi.GetDeviceByID) // 获取设备详情
		deviceRouterWithoutRecord.GET("stats", deviceApi.GetDeviceStats)     // 获取设备统计信息
	}
}