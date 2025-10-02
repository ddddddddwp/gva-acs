// Package factory provides factory methods for creating TR069 components.
package factory

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pool"
)

// PoolFactory 提供创建和管理各种对象池的工厂
type PoolFactory struct {
	bufferPool    *pool.BufferPool
	messagePool   *pool.MessagePool
	parameterPool *pool.ParameterPool
}

// NewPoolFactory 创建一个新的池工厂实例
func NewPoolFactory() *PoolFactory {
	return &PoolFactory{
		bufferPool:    pool.NewBufferPool(),
		messagePool:   pool.NewMessagePool(),
		parameterPool: pool.NewParameterPool(),
	}
}

// GetBufferPool 返回缓冲区池
func (f *PoolFactory) GetBufferPool() interfaces.BufferPool {
	return f.bufferPool
}

// GetMessagePool 返回消息对象池
func (f *PoolFactory) GetMessagePool() interfaces.MessagePool {
	return f.messagePool
}

// GetParameterPool 返回参数对象池
func (f *PoolFactory) GetParameterPool() interfaces.ParameterPool {
	return f.parameterPool
}
