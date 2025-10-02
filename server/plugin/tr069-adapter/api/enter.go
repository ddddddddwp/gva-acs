package api

type ApiGroup struct {
	DeviceApi
	ParameterApi
	SessionApi
	OperationLogApi
}

var ApiGroupApp = new(ApiGroup)