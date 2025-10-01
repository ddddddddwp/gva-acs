// Package version provides implementation for configuration version management functionality.
package version

import "github.com/root/demo/tr069/interfaces"

// NewVersionManagerFactory 创建配置版本管理器工厂函数
func NewVersionManagerFactory() func(options ...interfaces.VersionManagerOption) interfaces.ConfigVersionManager {
	return func(options ...interfaces.VersionManagerOption) interfaces.ConfigVersionManager {
		return NewVersionManager(options...)
	}
}
