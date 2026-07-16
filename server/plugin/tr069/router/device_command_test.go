package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeviceRouterRegistersAllTypedCommandRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	new(DeviceRouter).InitDeviceRouter(engine.Group("/tr069"))

	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		if route.Method == "POST" {
			routes[route.Path] = true
		}
	}

	want := []string{
		"/tr069/command/:deviceId/getRPCMethods",
		"/tr069/command/:deviceId/getParameterValues",
		"/tr069/command/:deviceId/getParameterNames",
		"/tr069/command/:deviceId/getParameterAttributes",
		"/tr069/command/:deviceId/setParameterValues",
		"/tr069/command/:deviceId/setParameterAttributes",
		"/tr069/command/:deviceId/addObject",
		"/tr069/command/:deviceId/deleteObject",
		"/tr069/command/:deviceId/download",
		"/tr069/command/:deviceId/upload",
		"/tr069/command/:deviceId/reboot",
		"/tr069/command/:deviceId/factoryReset",
	}

	for _, path := range want {
		if !routes[path] {
			t.Errorf("typed command route %q is not registered", path)
		}
	}
}
