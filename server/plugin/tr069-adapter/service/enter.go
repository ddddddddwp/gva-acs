package service

type ServiceGroup struct {
	DeviceService
	ParameterService
	SessionService
	TR069BridgeService
	TR069Service
}

var ServiceGroupApp = new(ServiceGroup)