// Package persistence provides implementation for configuration persistence functionality.
package persistence

import "github.com/root/demo/tr069/interfaces"

// WithPersistenceFormat 设置持久化格式
func WithPersistenceFormat(format interfaces.PersistenceFormat) interfaces.PersistenceOption {
return func(cp interfaces.ConfigPersistence) {
cp.SetFormat(format)
}
}
