package router

type RouterGroup struct {
	DeviceRouter
	ParameterRouter
	SessionRouter
	OperationLogRouter
	TR069Router
}

var RouterGroupApp = new(RouterGroup)