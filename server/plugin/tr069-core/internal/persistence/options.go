// Package persistence provides implementation for configuration persistence functionality.
package persistence

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// WithPersistenceFormat 设置持久化格式
func WithPersistenceFormat(format interfaces.PersistenceFormat) interfaces.PersistenceOption {
return func(cp interfaces.ConfigPersistence) {
cp.SetFormat(format)
}
}
