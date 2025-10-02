package api

import (
"context"
"encoding/json"
"fmt"
"net/http"
"time"

"github.com/gorilla/mux"
"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// Server 实现HTTP服务器
type Server struct {
router      *mux.Router
server      *http.Server
config      interfaces.APIConfig
middlewares []interfaces.Middleware
handlers    []interfaces.APIHandler
}

// NewServer 创建新的API服务器
func NewServer(options ...interfaces.APIOption) *Server {
// 默认配置
config := interfaces.APIConfig{
Port:            8080,
BasePath:        "/api/v1",
EnableCORS:      true,
EnableAuth:      false,
EnableRateLimit: false,
Timeout:         30,
EnableSwagger:   true,
}

// 应用选项
for _, option := range options {
option(&config)
}

router := mux.NewRouter()

// 创建服务器
server := &Server{
router: router,
config: config,
server: &http.Server{
Addr:         fmt.Sprintf(":%d", config.Port),
Handler:      router,
ReadTimeout:  time.Duration(config.Timeout) * time.Second,
WriteTimeout: time.Duration(config.Timeout) * time.Second,
},
}

// 设置基础路径
if config.BasePath != "" {
server.router = router.PathPrefix(config.BasePath).Subrouter()
}

return server
}

// RegisterHandler 注册API处理器
func (s *Server) RegisterHandler(handler interfaces.APIHandler) {
// 应用中间件
handler.SetMiddleware(s.middlewares...)

// 注册路由
handler.RegisterRoutes(s.routerAdapter())

// 保存处理器引用
s.handlers = append(s.handlers, handler)
}

// Use 添加中间件
func (s *Server) Use(middlewares ...interfaces.Middleware) {
s.middlewares = append(s.middlewares, middlewares...)

// 应用到已有处理器
for _, handler := range s.handlers {
handler.SetMiddleware(middlewares...)
}
}

// Start 启动服务器
func (s *Server) Start() error {
// 添加Swagger文档
if s.config.EnableSwagger {
s.setupSwagger()
}

// 添加CORS支持
if s.config.EnableCORS {
s.setupCORS()
}

// 启动服务器
return s.server.ListenAndServe()
}

// Stop 停止服务器
func (s *Server) Stop(ctx context.Context) error {
return s.server.Shutdown(ctx)
}

// routerAdapter 将mux路由器适配为接口
func (s *Server) routerAdapter() interfaces.Router {
return &routerAdapter{
router: s.router,
}
}

// setupSwagger 设置Swagger文档
func (s *Server) setupSwagger() {
// 简单实现，实际项目中可以使用swagger库
s.router.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]string{
"message": "Swagger documentation will be available here",
})
})
}

// setupCORS 设置CORS
func (s *Server) setupCORS() {
s.Use(func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return
}

next.ServeHTTP(w, r)
})
})
}

// routerAdapter 适配mux路由器到接口
type routerAdapter struct {
router *mux.Router
}

// Handle 实现Router接口
func (r *routerAdapter) Handle(method, path string, handler http.HandlerFunc) {
r.router.HandleFunc(path, handler).Methods(method)
}

// Group 实现Router接口
func (r *routerAdapter) Group(prefix string) interfaces.Router {
return &routerAdapter{
router: r.router.PathPrefix(prefix).Subrouter(),
}
}

// Use 实现Router接口
func (r *routerAdapter) Use(middlewares ...interfaces.Middleware) {
for _, middleware := range middlewares {
r.router.Use(mux.MiddlewareFunc(middleware))
}
}
