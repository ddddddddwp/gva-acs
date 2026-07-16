package utils

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// SystemEvents 定义系统级事件处理
type SystemEvents struct {
	reloadHandlers       []func() error
	configChangeHandlers []func()
	shutdownHandlers     []func(context.Context) error
	mu                   sync.RWMutex
}

// RegisterShutdownHandler 注册在 HTTP 服务关闭期间运行的清理函数。
func (e *SystemEvents) RegisterShutdownHandler(handler func(context.Context) error) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shutdownHandlers = append(e.shutdownHandlers, handler)
}

// 全局事件管理器
var GlobalSystemEvents = &SystemEvents{}

// RegisterReloadHandler 注册系统重载处理函数
func (e *SystemEvents) RegisterReloadHandler(handler func() error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reloadHandlers = append(e.reloadHandlers, handler)
}

// RegisterConfigChangeHandler 注册轻量级配置变更处理函数。
func (e *SystemEvents) RegisterConfigChangeHandler(handler func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.configChangeHandlers = append(e.configChangeHandlers, handler)
}

// TriggerConfigChange 使用处理函数快照触发轻量级配置热更新。
func (e *SystemEvents) TriggerConfigChange() {
	e.mu.RLock()
	handlers := append([]func(){}, e.configChangeHandlers...)
	e.mu.RUnlock()

	for _, handler := range handlers {
		handler()
	}
}

// TriggerShutdown 使用处理函数快照执行所有清理函数，并保留每个错误。
func (e *SystemEvents) TriggerShutdown(ctx context.Context) error {
	if e == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	e.mu.RLock()
	handlers := append([]func(context.Context) error{}, e.shutdownHandlers...)
	e.mu.RUnlock()

	var shutdownErr error
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		shutdownErr = errors.Join(shutdownErr, invokeShutdownHandler(ctx, handler))
	}
	return shutdownErr
}

func invokeShutdownHandler(ctx context.Context, handler func(context.Context) error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errors.Join(err, fmt.Errorf("shutdown handler panic: %v", recovered))
		}
	}()
	return handler(ctx)
}

// TriggerReload 触发所有注册的重载处理函数
func (e *SystemEvents) TriggerReload() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, handler := range e.reloadHandlers {
		if err := handler(); err != nil {
			return err
		}
	}
	return nil
}
