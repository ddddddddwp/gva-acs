package rest

import (
	"net/http"
	"time"

	"github.com/root/demo/tr069/interfaces"
)

// CORSMiddleware 创建CORS中间件
func CORSMiddleware() interfaces.Middleware {
	return func(next http.Handler) http.Handler {
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
	}
}

// LoggingMiddleware 创建日志中间件
func LoggingMiddleware() interfaces.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// 包装ResponseWriter以捕获状态码
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			next.ServeHTTP(wrapper, r)
			
			// 记录请求信息
			duration := time.Since(start)
			
			// 这里可以使用项目的日志系统记录
			// 简单起见，使用标准输出
			println(r.Method, r.URL.Path, wrapper.statusCode, duration.String())
		})
	}
}

// AuthMiddleware 创建认证中间件
func AuthMiddleware(authType string) interfaces.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 根据不同的认证类型实现不同的认证逻辑
			switch authType {
			case "basic":
				username, password, ok := r.BasicAuth()
				if !ok || !validateBasicAuth(username, password) {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
					SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized", 401)
					return
				}
			case "token":
				token := r.Header.Get("Authorization")
				if !validateToken(token) {
					SendErrorResponse(w, http.StatusUnauthorized, "Invalid token", 401)
					return
				}
			default:
				// 默认不做认证
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware 创建速率限制中间件
func RateLimitMiddleware(limit int) interfaces.Middleware {
	// 简单实现，实际应用中应使用更复杂的速率限制算法
	var requestCount = make(map[string]int)
	var lastReset = time.Now()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 每分钟重置计数
			if time.Since(lastReset) > time.Minute {
				requestCount = make(map[string]int)
				lastReset = time.Now()
			}
			
			// 使用IP作为标识
			ip := r.RemoteAddr
			
			// 检查请求数
			if requestCount[ip] >= limit {
				SendErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded", 429)
				return
			}
			
			// 增加计数
			requestCount[ip]++
			
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriterWrapper 包装http.ResponseWriter以捕获状态码
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// 验证基本认证
func validateBasicAuth(username, password string) bool {
	// 实际应用中应从配置或数据库中获取凭据
	return username == "admin" && password == "password"
}

// 验证令牌
func validateToken(token string) bool {
	// 实际应用中应验证JWT或其他令牌
	return len(token) > 7 && token[:7] == "Bearer "
}