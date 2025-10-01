// Package rpc provides custom RPC method registration and handling.
package rpc

import (
	"context"
	"fmt"
	"sync"
)

// HandlerFunc defines the signature for RPC method handlers.
type HandlerFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// Registry manages custom RPC method registrations.
type Registry struct {
	methods map[string]HandlerFunc
	mutex   sync.RWMutex
}

// NewRegistry creates a new RPC method registry.
func NewRegistry() *Registry {
	return &Registry{
		methods: make(map[string]HandlerFunc),
	}
}

// Register registers a custom RPC method handler.
func (r *Registry) Register(method string, handler HandlerFunc) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Check if method already exists
	if _, exists := r.methods[method]; exists {
		return fmt.Errorf("method %s already registered", method)
	}
	
	r.methods[method] = handler
	return nil
}

// Unregister removes a custom RPC method handler.
func (r *Registry) Unregister(method string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Check if method exists
	if _, exists := r.methods[method]; !exists {
		return fmt.Errorf("method %s not found", method)
	}
	
	delete(r.methods, method)
	return nil
}

// Handle processes a custom RPC method call.
func (r *Registry) Handle(ctx context.Context, method string, params map[string]interface{}) (interface{}, error) {
	r.mutex.RLock()
	handler, exists := r.methods[method]
	r.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("method %s not registered", method)
	}
	
	return handler(ctx, params)
}

// ListMethods returns a list of all registered custom methods.
func (r *Registry) ListMethods() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	methods := make([]string, 0, len(r.methods))
	for method := range r.methods {
		methods = append(methods, method)
	}
	
	return methods
}

// IsRegistered checks if a method is registered.
func (r *Registry) IsRegistered(method string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	_, exists := r.methods[method]
	return exists
}