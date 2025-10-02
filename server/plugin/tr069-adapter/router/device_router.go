package router

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/api"
)

type DeviceRouter struct{}

// InitDeviceRouter 初始化设备路由
func (d *DeviceRouter) InitDeviceRouter(Router *gin.RouterGroup) {
	deviceRouter := Router.Group("device").Use(middleware.OperationRecord())
	deviceRouterWithoutRecord := Router.Group("device")
	deviceApi := api.ApiGroupApp.DeviceApi

	{
		// 需要记录操作的路由
		deviceRouter.POST("createDevice", deviceApi.CreateDevice)                     // 创建设备
		deviceRouter.PUT("updateDevice", deviceApi.UpdateDevice)                     // 更新设备
		deviceRouter.DELETE("deleteDevice/:id", deviceApi.DeleteDevice)             // 删除设备
		deviceRouter.POST("batchDeleteDevices", deviceApi.BatchDeleteDevices)       // 批量删除设备
		deviceRouter.POST("rebootDevice/:id", deviceApi.RebootDevice)               // 重启设备
		deviceRouter.POST("factoryResetDevice/:id", deviceApi.FactoryResetDevice)   // 恢复出厂设置
	}

	{
		// 不需要记录操作的路由
		deviceRouterWithoutRecord.GET("list", deviceApi.GetDeviceList)                              // 获取设备列表
		deviceRouterWithoutRecord.POST("getDeviceList", deviceApi.GetDeviceList)                    // 获取设备列表（兼容POST）
		deviceRouterWithoutRecord.GET("getDeviceByID/:id", deviceApi.GetDeviceByID)                 // 根据ID获取设备
		deviceRouterWithoutRecord.GET("getDeviceBySerialNumber/:sn", deviceApi.GetDeviceBySerialNumber) // 根据序列号获取设备
		deviceRouterWithoutRecord.GET("getDeviceStats", deviceApi.GetDeviceStats)                   // 获取设备统计信息
	}
}