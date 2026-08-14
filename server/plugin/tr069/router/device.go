package router

import (
	"github.com/ddddddddwp/gva-acs/server/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/api"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/redact"
	"github.com/gin-gonic/gin"
)

type DeviceRouter struct {
	deviceApi *api.DeviceApi
}

func NewDeviceRouter(deviceApi *api.DeviceApi) *DeviceRouter {
	return &DeviceRouter{deviceApi: deviceApi}
}

func (r *DeviceRouter) InitDeviceRouter(Router *gin.RouterGroup) {
	// Device Base Management
	deviceRouter := Router.Group("device")
	deviceApi := r.deviceApi
	if deviceApi == nil {
		deviceApi = api.NewDeviceApi(nil)
	}
	{
		deviceRouter.GET("list", deviceApi.GetDeviceList)
		deviceRouter.POST("", deviceApi.CreateDevice)
		deviceRouter.DELETE(":deviceId", deviceApi.DeleteDevice)
		deviceRouter.GET(":deviceId/connection-profile", deviceApi.GetConnectionProfile)
		deviceRouter.PUT(":deviceId/connection-profile", middleware.OperationRecordWithBodySanitizer(redact.ConnectionProfileJSON), deviceApi.UpdateConnectionProfile)
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
		debugRouter.GET("trace/:traceId", debugApi.GetTrace)
	}

	commandRouter := Router.Group("command").Use(middleware.OperationRecordWithBodySanitizer(redact.CommandJSON))
	commandApi := new(api.CommandApi)
	{
		commandRouter.POST(":deviceId/getRPCMethods", commandApi.SyncRPCMethods)
		commandRouter.POST(":deviceId/getParameterValues", commandApi.GetParameterValues)
		commandRouter.POST(":deviceId/getParameterNames", commandApi.GetParameterNames)
		commandRouter.POST(":deviceId/getParameterAttributes", commandApi.GetParameterAttributes)
		commandRouter.POST(":deviceId/setParameterValues", commandApi.SetParameterValues)
		commandRouter.POST(":deviceId/setParameterAttributes", commandApi.SetParameterAttributes)
		commandRouter.POST(":deviceId/addObject", commandApi.AddObject)
		commandRouter.POST(":deviceId/deleteObject", commandApi.DeleteObject)
		commandRouter.POST(":deviceId/download", commandApi.Download)
		commandRouter.POST(":deviceId/upload", commandApi.Upload)
		commandRouter.POST(":deviceId/reboot", commandApi.Reboot)
		commandRouter.POST(":deviceId/factoryReset", commandApi.FactoryReset)
	}

	recordRouter := Router.Group("command-record")
	recordApi := new(api.CommandRecordApi)
	{
		recordRouter.GET("list", recordApi.List)
		recordRouter.GET(":commandId", recordApi.Detail)
		recordRouter.POST(":commandId/retry", middleware.OperationRecord(), recordApi.Retry)
	}

	dmRouter := Router.Group("datamodel")
	dmApi := new(api.DataModelApi)
	{
		dmRouter.POST(":deviceId/sync", dmApi.FullSync)
		dmRouter.GET(":deviceId/list", dmApi.GetDataModelList)
		dmRouter.GET(":deviceId/structure", dmApi.GetDataModelStructure)
	}
}
