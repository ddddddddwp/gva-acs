package service

// ServiceGroup 服务组
type ServiceGroup struct {
	DeviceService
	ParameterService
	AdapterService
}

var ServiceGroupApp = new(ServiceGroup)