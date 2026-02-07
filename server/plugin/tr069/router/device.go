package router

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/api"
	"github.com/gin-gonic/gin"
)

type DeviceRouter struct{}

func (r *DeviceRouter) InitDeviceRouter(Router *gin.RouterGroup) {
	deviceRouter := Router.Group("tr069/device")
	deviceApi := new(api.DeviceApi)
	{
		deviceRouter.GET("list", deviceApi.GetDeviceList)
	}
}
