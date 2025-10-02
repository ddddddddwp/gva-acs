package router

type RouterGroup struct {
	DeviceRouter
	ParameterRouter
	SessionRouter
	OperationLogRouter
}

var RouterGroupApp = new(RouterGroup)