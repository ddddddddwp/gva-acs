package service

import (
	"context"
	"errors"
)

var ErrDeviceDeleting = errors.New("device is being deleted")

type DeviceRuntimeIdentity struct {
	DeviceID  uint
	DeviceKey string
	IP        string
}

type DeviceRuntimeCleaner interface {
	Purge(context.Context, DeviceRuntimeIdentity) error
}
