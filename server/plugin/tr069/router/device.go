package router

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/api"
	"github.com/gin-gonic/gin"
)

type DeviceRouter struct{}

func (r *DeviceRouter) InitDeviceRouter(Router *gin.RouterGroup) {
	// Device Base Management
	deviceRouter := Router.Group("device")
	deviceApi := new(api.DeviceApi)
	{
		deviceRouter.GET("list", deviceApi.GetDeviceList)
		deviceRouter.POST("", deviceApi.CreateDevice)
		deviceRouter.DELETE(":deviceId", deviceApi.DeleteDevice)
	}

	// FAP (Base Station) Specific Management
	fapRouter := Router.Group("fap")
	fapApi := new(api.FAPApi)
	{
		fapRouter.GET(":deviceId", fapApi.GetFAPInfo)
		fapRouter.POST(":deviceId/sync", fapApi.SyncFAPInfo)
		fapRouter.PUT(":deviceId", fapApi.ConfigureFAP)
	}

	debugRouter := Router.Group("debug")
	debugApi := new(api.DebugApi)
	{
		debugRouter.GET("trace/:requestId", debugApi.GetTrace)
	}

	commandRouter := Router.Group("command")
	commandApi := new(api.CommandApi)
	{
		commandRouter.POST(":deviceId/getRPCMethods", commandApi.SyncRPCMethods)
		commandRouter.POST(":deviceId/getParameterValues", commandApi.GetParameterValues)
		commandRouter.POST(":deviceId/setParameterValues", commandApi.SetParameterValues)
	}

	dmRouter := Router.Group("datamodel")
	dmApi := new(api.DataModelApi)
	{
		dmRouter.POST(":deviceId/sync", dmApi.FullSync)
	}
}
