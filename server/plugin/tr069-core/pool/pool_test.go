// Package pool provides object pooling for the TR069 library.
package pool

import (
 "testing"
 
 "github.com/root/demo/tr069/interfaces"
)

func TestBufferPool(t *testing.T) {
 pool := NewBufferPool()
 
 // Get a buffer
 buf := pool.Get()
 if buf == nil {
  t.Fatal("Expected a buffer, got nil")
 }
 
 // Check that it's empty
 if buf.Len() != 0 {
  t.Errorf("Expected empty buffer, got length %d", buf.Len())
 }
 
 // Write some data to it
 buf.WriteString("test data")
 
 // Put it back
 pool.Put(buf)
 
 // Get it again
 buf2 := pool.Get()
 if buf2 == nil {
  t.Fatal("Expected a buffer, got nil")
 }
 
 // Check that it's empty again (reset)
 if buf2.Len() != 0 {
  t.Errorf("Expected empty buffer after reset, got length %d", buf2.Len())
 }
}

func TestMessagePool(t *testing.T) {
 pool := NewMessagePool()
 
 // Get a message
 msg := pool.Get()
 if msg == nil {
  t.Fatal("Expected a message, got nil")
 }
 
 // Check that fields are reset
 if msg.Method != "" {
  t.Errorf("Expected empty method, got %s", msg.Method)
 }
 
 if msg.Parameters != nil && len(msg.Parameters) != 0 {
  t.Errorf("Expected empty parameters, got %d", len(msg.Parameters))
 }
 
 if msg.Fault != nil {
  t.Errorf("Expected nil fault, got %v", msg.Fault)
 }
 
 // Set some values
 msg.Method = "test"
 msg.Parameters = append(msg.Parameters, interfaces.Parameter{Name: "test"})
 msg.Fault = &interfaces.Fault{FaultCode: 100}
 
 // Put it back
 pool.Put(msg)
 
 // Get it again
 msg2 := pool.Get()
 if msg2 == nil {
  t.Fatal("Expected a message, got nil")
 }
 
 // Check that fields are reset again
 if msg2.Method != "" {
  t.Errorf("Expected empty method after reset, got %s", msg2.Method)
 }
 
 if msg2.Parameters != nil && len(msg2.Parameters) != 0 {
  t.Errorf("Expected empty parameters after reset, got %d", len(msg2.Parameters))
 }
 
 if msg2.Fault != nil {
  t.Errorf("Expected nil fault after reset, got %v", msg2.Fault)
 }
}

func TestParameterPool(t *testing.T) {
 pool := NewParameterPool()
 
 // Get a parameter
 param := pool.Get()
 if param == nil {
  t.Fatal("Expected a parameter, got nil")
 }
 
 // Check that fields are reset
 if param.Name != "" {
  t.Errorf("Expected empty name, got %s", param.Name)
 }
 
 if param.Value != nil {
  t.Errorf("Expected nil value, got %v", param.Value)
 }
 
 if param.Type != "" {
  t.Errorf("Expected empty type, got %s", param.Type)
 }
 
 // Set some values
 param.Name = "test"
 param.Value = "value"
 param.Type = "string"
 
 // Put it back
 pool.Put(param)
 
 // Get it again
 param2 := pool.Get()
 if param2 == nil {
  t.Fatal("Expected a parameter, got nil")
 }
 
 // Check that fields are reset again
 if param2.Name != "" {
  t.Errorf("Expected empty name after reset, got %s", param2.Name)
 }
 
 if param2.Value != nil {
  t.Errorf("Expected nil value after reset, got %v", param2.Value)
 }
 
 if param2.Type != "" {
  t.Errorf("Expected empty type after reset, got %s", param2.Type)
 }
}