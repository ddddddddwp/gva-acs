// Package config provides configuration updater functionality.
package config

import (
	"sync"

	"github.com/root/demo/tr069/interfaces"
)

// ConfigUpdater handles configuration updates.
type ConfigUpdater struct {
	mu       sync.RWMutex
	config   interfaces.Config
	listener *ConfigListener
}

// NewConfigUpdater creates a new configuration updater.
func NewConfigUpdater(config interfaces.Config) *ConfigUpdater {
	return &ConfigUpdater{
		config:   config,
		listener: NewConfigListener(),
	}
}

// UpdateStrictMode updates the strict mode setting.
func (cu *ConfigUpdater) UpdateStrictMode(strict bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	
	oldValue := cu.config.GetStrictMode()
	cu.config.SetStrictMode(strict)
	cu.listener.NotifyChange("strict_mode", oldValue, strict)
}

// UpdatePrettyPrint updates the pretty print setting.
func (cu *ConfigUpdater) UpdatePrettyPrint(pretty bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	
	oldValue := cu.config.GetPrettyPrint()
	cu.config.SetPrettyPrint(pretty)
	cu.listener.NotifyChange("pretty_print", oldValue, pretty)
}

// UpdateMaxDepth updates the maximum depth setting.
func (cu *ConfigUpdater) UpdateMaxDepth(depth int) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	
	oldValue := cu.config.GetMaxDepth()
	cu.config.SetMaxDepth(depth)
	cu.listener.NotifyChange("max_depth", oldValue, depth)
}

// UpdateValidation updates the validation setting.
func (cu *ConfigUpdater) UpdateValidation(enabled bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	
	oldValue := cu.config.GetValidation()
	cu.config.SetValidation(enabled)
	cu.listener.NotifyChange("validation", oldValue, enabled)
}

// GetConfig returns the current configuration.
func (cu *ConfigUpdater) GetConfig() interfaces.Config {
	cu.mu.RLock()
	defer cu.mu.RUnlock()
	return cu.config
}

// AddListener adds a configuration change listener.
func (cu *ConfigUpdater) AddListener(listener interfaces.ConfigChangeListener) {
	cu.listener.AddListener(listener)
}

// RemoveListener removes a configuration change listener.
func (cu *ConfigUpdater) RemoveListener(listener interfaces.ConfigChangeListener) {
	cu.listener.RemoveListener(listener)
}