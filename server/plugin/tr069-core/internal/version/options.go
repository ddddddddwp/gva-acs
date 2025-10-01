// Package version provides implementation for configuration version management functionality.
package version

import "github.com/root/demo/tr069/interfaces"

// WithVersionStoragePath 设置版本存储路径
func WithVersionStoragePath(path string) interfaces.VersionManagerOption {
	return func(vm interfaces.ConfigVersionManager) {
		vm.SetVersionStoragePath(path)
	}
}
