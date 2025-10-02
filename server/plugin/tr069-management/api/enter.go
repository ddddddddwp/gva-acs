package api

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/service"

// ApiGroup API组
type ApiGroup struct {
	DeviceApi
	ParameterApi
	AdapterApi
}

var (
	ApiGroupApp = new(ApiGroup)
	deviceService = service.ServiceGroupApp.DeviceService
	parameterService = service.ServiceGroupApp.ParameterService
	adapterService = service.ServiceGroupApp.AdapterService
)