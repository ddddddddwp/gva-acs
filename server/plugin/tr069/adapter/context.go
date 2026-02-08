package adapter

import (
	"context"

	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
)

type ctxKey int

const (
	ctxKeyDeviceID ctxKey = iota
	ctxKeyClientIP
)

func WithDeviceMeta(ctx context.Context, deviceID *tr069.DeviceID, clientIP string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if deviceID != nil {
		ctx = context.WithValue(ctx, ctxKeyDeviceID, deviceID)
	}
	if clientIP != "" {
		ctx = context.WithValue(ctx, ctxKeyClientIP, clientIP)
	}
	return ctx
}

func deviceIDFromContext(ctx context.Context) (*tr069.DeviceID, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(ctxKeyDeviceID)
	if v == nil {
		return nil, false
	}
	id, ok := v.(*tr069.DeviceID)
	return id, ok && id != nil
}

func clientIPFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(ctxKeyClientIP)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}
