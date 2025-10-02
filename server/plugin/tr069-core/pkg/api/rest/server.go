package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// Server 实现RESTful API服务器
type Server struct {
	config     *ServerConfig
	router     http.Handler
	middleware []interfaces.Middleware
	handlers   []interfaces.APIHandler
}

// ServerConfig 定义服务器配置
type ServerConfig struct {
	Port            int
	BasePath        string
	EnableCORS      bool
	EnableAuth      bool
	AuthType        string
	EnableRateLimit bool
	RateLimit       int
	Timeout         int // 秒
	EnableSwagger   bool
}

// NewServer 创建新的RESTful API服务器
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{
			Port:      8080,
			BasePath:  "/api/v1",
			Timeout:   30,
			EnableCORS: true,
		}
	}
	
	return &Server{
		config:     config,
		middleware: make([]interfaces.Middleware, 0),
		handlers:   make([]interfaces.APIHandler, 0),
	}
}

// WithMiddleware 添加中间件
func (s *Server) WithMiddleware(middleware ...interfaces.Middleware) *Server {
	s.middleware = append(s.middleware, middleware...)
	return s
}

// RegisterHandler 注册API处理器
func (s *Server) RegisterHandler(handler interfaces.APIHandler) *Server {
	s.handlers = append(s.handlers, handler)
	return s
}

// Start 启动服务器
func (s *Server) Start() error {
	router := NewRouter(s.config.BasePath)
	
	// 应用全局中间件
	for _, m := range s.middleware {
		router.Use(m)
	}
	
	// 注册所有处理器的路由
	for _, h := range s.handlers {
		h.RegisterRoutes(router)
	}
	
	s.router = router.Handler()
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Timeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Timeout) * time.Second,
	}
	
	fmt.Printf("RESTful API server starting on port %d...\n", s.config.Port)
	return server.ListenAndServe()
}

// SendJSONResponse 发送JSON响应
func SendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// SendSuccessResponse 发送成功响应
func SendSuccessResponse(w http.ResponseWriter, data interface{}) {
	response := interfaces.APIResponse{
		Success: true,
		Data:    data,
	}
	SendJSONResponse(w, http.StatusOK, response)
}

// SendErrorResponse 发送错误响应
func SendErrorResponse(w http.ResponseWriter, status int, message string, code int) {
	response := interfaces.APIResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}
	SendJSONResponse(w, status, response)
}