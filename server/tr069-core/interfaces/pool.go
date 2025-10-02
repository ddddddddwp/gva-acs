// Package interfaces defines the public interfaces for the TR069 library.
package interfaces

import "bytes"

// BufferPool 定义了缓冲区对象池的接口
type BufferPool interface {
	// Get 从对象池获取一个缓冲区
	Get() *bytes.Buffer

	// Put 将缓冲区放回对象池
	Put(*bytes.Buffer)
}

// MessagePool 定义了消息对象池的接口
type MessagePool interface {
	// Get 从对象池获取一个消息对象
	Get() *Message

	// Put 将消息对象放回对象池
	Put(*Message)
}

// ParameterPool 定义了参数对象池的接口
type ParameterPool interface {
	// Get 从对象池获取一个参数对象
	Get() *Parameter

	// Put 将参数对象放回对象池
	Put(*Parameter)
}
