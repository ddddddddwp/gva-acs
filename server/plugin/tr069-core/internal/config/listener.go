// Package config provides configuration listener functionality.
package config

import (
	"sync"

	"github.com/root/demo/tr069/interfaces"
)

// ConfigListener listens for configuration changes.
type ConfigListener struct {
	mu        sync.RWMutex
	listeners []interfaces.ConfigChangeListener
}

// NewConfigListener creates a new configuration listener.
func NewConfigListener() *ConfigListener {
	return &ConfigListener{
		listeners: make([]interfaces.ConfigChangeListener, 0),
	}
}

// AddListener adds a configuration change listener.
func (cl *ConfigListener) AddListener(listener interfaces.ConfigChangeListener) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.listeners = append(cl.listeners, listener)
}

// RemoveListener removes a configuration change listener.
func (cl *ConfigListener) RemoveListener(listener interfaces.ConfigChangeListener) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	for i, l := range cl.listeners {
		if l == listener {
			cl.listeners = append(cl.listeners[:i], cl.listeners[i+1:]...)
			break
		}
	}
}

// NotifyChange notifies all listeners of a configuration change.
func (cl *ConfigListener) NotifyChange(key string, oldValue, newValue interface{}) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	
	for _, listener := range cl.listeners {
		go listener.OnConfigChange(key, oldValue, newValue)
	}
}