package service

type ServiceGroup struct {
	DeviceService
	ParameterService
	SessionService
	TR069BridgeService
}

var ServiceGroupApp = new(ServiceGroup)