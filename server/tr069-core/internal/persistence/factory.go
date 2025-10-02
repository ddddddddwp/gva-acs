// Package persistence provides implementation for configuration persistence functionality.
package persistence

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// NewPersistenceFactory 创建配置持久化工厂函数
func NewPersistenceFactory() func(options ...interfaces.PersistenceOption) interfaces.ConfigPersistence {
return func(options ...interfaces.PersistenceOption) interfaces.ConfigPersistence {
return NewConfigPersistence(options...)
}
}
