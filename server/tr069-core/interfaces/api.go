package interfaces

import (
	"net/http"
)

// APIHandler 定义RESTful API处理接口
type APIHandler interface {
	// RegisterRoutes 注册API路由
	RegisterRoutes(router Router)

	// SetMiddleware 设置中间件
	SetMiddleware(middleware ...Middleware)
}

// Router 定义路由接口
type Router interface {
	// Handle 注册HTTP方法和路径的处理函数
	Handle(method, path string, handler http.HandlerFunc)

	// Group 创建路由组
	Group(prefix string) Router

	// Use 使用中间件
	Use(middleware ...Middleware)
}

// Middleware 定义中间件接口
type Middleware func(http.Handler) http.Handler

// APIResponse 定义API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code,omitempty"`
}

// APIOption 定义API配置选项函数类型
type APIOption func(*APIConfig)

// APIConfig 定义API配置
type APIConfig struct {
	Port            int
	BasePath        string
	EnableCORS      bool
	EnableAuth      bool
	AuthType        string
	EnableRateLimit bool
	RateLimit       int
	Timeout         int
	EnableSwagger   bool
}
