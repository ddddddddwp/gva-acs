// Package version provides implementation for configuration version management functionality.
package version

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// WithVersionStoragePath 设置版本存储路径
func WithVersionStoragePath(path string) interfaces.VersionManagerOption {
	return func(vm interfaces.ConfigVersionManager) {
		vm.SetVersionStoragePath(path)
	}
}
