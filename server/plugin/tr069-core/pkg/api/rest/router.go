package rest

import (
	"net/http"
	"path"
	"strings"

	"github.com/root/demo/tr069/interfaces"
)

// Router 实现路由接口
type Router struct {
	basePath   string
	mux        *http.ServeMux
	middleware []interfaces.Middleware
	routes     map[string]http.Handler
}

// NewRouter 创建新的路由器
func NewRouter(basePath string) *Router {
	return &Router{
		basePath:   basePath,
		mux:        http.NewServeMux(),
		middleware: make([]interfaces.Middleware, 0),
		routes:     make(map[string]http.Handler),
	}
}

// Handle 注册HTTP方法和路径的处理函数
func (r *Router) Handle(method, pathPattern string, handler http.HandlerFunc) {
	// 构建完整路径
	fullPath := path.Join(r.basePath, pathPattern)
	
	// 创建方法过滤处理器
	methodHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			w.Header().Set("Allow", method)
			SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", 405)
			return
		}
		handler(w, req)
	})
	
	// 应用中间件
	finalHandler := methodHandler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		finalHandler = r.middleware[i](finalHandler).(http.HandlerFunc)
	}
	
	// 注册到路由表
	r.routes[fullPath] = finalHandler
	r.mux.Handle(fullPath+"/", http.StripPrefix(fullPath, finalHandler))
	r.mux.Handle(fullPath, finalHandler)
}

// Group 创建路由组
func (r *Router) Group(prefix string) interfaces.Router {
	return &Router{
		basePath:   path.Join(r.basePath, prefix),
		mux:        r.mux,
		middleware: append([]interfaces.Middleware{}, r.middleware...),
		routes:     r.routes,
	}
}

// Use 使用中间件
func (r *Router) Use(middleware ...interfaces.Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

// Handler 返回HTTP处理器
func (r *Router) Handler() http.Handler {
	return r.mux
}

// PathMatch 检查路径是否匹配
func PathMatch(pattern, path string) bool {
	if pattern == path {
		return true
	}
	
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	
	return path == pattern || path == pattern+"/" || strings.HasPrefix(path, pattern+"/")
}