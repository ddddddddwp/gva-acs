// Package interfaces defines the public interfaces for the TR069 library.
package interfaces

// ConfigUpdateEvent 配置更新事件类型
type ConfigUpdateEvent string

const (
	// ConfigUpdateEventAdd 添加配置项事件
	ConfigUpdateEventAdd ConfigUpdateEvent = "add"
	// ConfigUpdateEventModify 修改配置项事件
	ConfigUpdateEventModify ConfigUpdateEvent = "modify"
	// ConfigUpdateEventDelete 删除配置项事件
	ConfigUpdateEventDelete ConfigUpdateEvent = "delete"
	// ConfigUpdateEventReload 重新加载配置事件
	ConfigUpdateEventReload ConfigUpdateEvent = "reload"
)

// ConfigChangeListener 配置变更监听器接口
type ConfigChangeListener interface {
	// OnConfigChange 当配置变更时调用
	OnConfigChange(key string, oldValue, newValue interface{})
}

// ConfigUpdater 配置更新器接口
type ConfigUpdater interface {
	// UpdateConfig 更新配置项
	UpdateConfig(key string, value interface{}) error

	// DeleteConfig 删除配置项
	DeleteConfig(key string) error

	// ReloadConfig 重新加载配置
	ReloadConfig() error

	// AddListener 添加配置更新监听器
	AddListener(listener ConfigChangeListener)

	// RemoveListener 移除配置更新监听器
	RemoveListener(listener ConfigChangeListener)

	// GetConfig 获取配置项
	GetConfig(key string) (interface{}, error)

	// GetAllConfig 获取所有配置项
	GetAllConfig() map[string]interface{}

	// Start 启动配置更新器
	Start() error

	// Stop 停止配置更新器
	Stop() error
}

// ConfigUpdaterOption 配置更新器选项函数
type ConfigUpdaterOption func(ConfigUpdater)

// WithConfigSource 设置配置源
func WithConfigSource(source string) ConfigUpdaterOption {
	return func(cu ConfigUpdater) {
		// 具体实现在内部包中
	}
}

// WithConfigFormat 设置配置格式
func WithConfigFormat(format string) ConfigUpdaterOption {
	return func(cu ConfigUpdater) {
		// 具体实现在内部包中
	}
}

// WithConfigWatchInterval 设置配置监视间隔
func WithConfigWatchInterval(intervalSeconds int) ConfigUpdaterOption {
	return func(cu ConfigUpdater) {
		// 具体实现在内部包中
	}
}