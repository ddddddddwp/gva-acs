// Package pool provides object pooling for the TR069 library.
package pool

import (
 "bytes"
 "sync"
 
 "github.com/root/demo/tr069/interfaces"
)

// BufferPool is a pool of bytes.Buffer objects.
type BufferPool struct {
 pool sync.Pool
}

// NewBufferPool creates a new BufferPool.
func NewBufferPool() *BufferPool {
 return &BufferPool{
  pool: sync.Pool{
   New: func() interface{} {
    return &bytes.Buffer{}
   },
  },
 }
}

// Get returns a bytes.Buffer from the pool.
func (p *BufferPool) Get() *bytes.Buffer {
 buf := p.pool.Get().(*bytes.Buffer)
 buf.Reset()
 return buf
}

// Put returns a bytes.Buffer to the pool.
func (p *BufferPool) Put(buf *bytes.Buffer) {
 p.pool.Put(buf)
}

// MessagePool is a pool of interfaces.Message objects.
type MessagePool struct {
 pool sync.Pool
}

// NewMessagePool creates a new MessagePool.
func NewMessagePool() *MessagePool {
 return &MessagePool{
  pool: sync.Pool{
   New: func() interface{} {
    return &interfaces.Message{}
   },
  },
 }
}

// Get returns a Message from the pool.
func (p *MessagePool) Get() *interfaces.Message {
 msg := p.pool.Get().(*interfaces.Message)
 // Reset the message fields
 msg.Method = ""
 msg.Parameters = msg.Parameters[:0]
 msg.Fault = nil
 msg.SessionID = ""
 msg.HoldRequests = false
 msg.NoMoreRequests = false
 return msg
}

// Put returns a Message to the pool.
func (p *MessagePool) Put(msg *interfaces.Message) {
 p.pool.Put(msg)
}

// ParameterPool is a pool of interfaces.Parameter objects.
type ParameterPool struct {
 pool sync.Pool
}

// NewParameterPool creates a new ParameterPool.
func NewParameterPool() *ParameterPool {
 return &ParameterPool{
  pool: sync.Pool{
   New: func() interface{} {
    return &interfaces.Parameter{}
   },
  },
 }
}

// Get returns a Parameter from the pool.
func (p *ParameterPool) Get() *interfaces.Parameter {
 param := p.pool.Get().(*interfaces.Parameter)
 // Reset the parameter fields
 param.Name = ""
 param.Value = nil
 param.Type = ""
 return param
}

// Put returns a Parameter to the pool.
func (p *ParameterPool) Put(param *interfaces.Parameter) {
 p.pool.Put(param)
}

// Global pools
var (
 Buffers   = NewBufferPool()
 Messages  = NewMessagePool()
 Parameters = NewParameterPool()
)