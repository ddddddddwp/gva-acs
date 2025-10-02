package api

type ApiGroup struct {
	DeviceApi
	ParameterApi
	SessionApi
	OperationLogApi
	TR069Api
}

var ApiGroupApp = new(ApiGroup)