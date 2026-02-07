package service

import (
	"github.com/ddddddddwp/gva-acs/server/service/example"
	"github.com/ddddddddwp/gva-acs/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup  system.ServiceGroup
	ExampleServiceGroup example.ServiceGroup
}
