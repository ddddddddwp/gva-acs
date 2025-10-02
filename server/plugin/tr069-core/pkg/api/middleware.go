package api

import (
"context"
"net/http"
"strings"
"time"

"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// 常用中间件实现

// LoggingMiddleware 创建日志中间件
func LoggingMiddleware(logger interfaces.Logger) interfaces.Middleware {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()

// 包装ResponseWriter以捕获状态码
rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

// 处理请求
next.ServeHTTP(rw, r)

// 记录请求信息
		duration := time.Since(start)
		logger.Info("API Request",
			interfaces.LogField{Key: "method", Value: r.Method},
			interfaces.LogField{Key: "path", Value: r.URL.Path},
			interfaces.LogField{Key: "status", Value: rw.statusCode},
			interfaces.LogField{Key: "duration", Value: duration.String()},
			interfaces.LogField{Key: "ip", Value: r.RemoteAddr},
			interfaces.LogField{Key: "user_agent", Value: r.UserAgent()},
		)
})
}
}

// AuthMiddleware 创建认证中间件
func AuthMiddleware(authType string, validator func(string) bool) interfaces.Middleware {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 获取认证信息
authHeader := r.Header.Get("Authorization")

// 检查认证头
if authHeader == "" {
http.Error(w, "Unauthorized: No authentication provided", http.StatusUnauthorized)
return
}

// 验证认证类型
parts := strings.SplitN(authHeader, " ", 2)
if len(parts) != 2 || !strings.EqualFold(parts[0], authType) {
http.Error(w, "Unauthorized: Invalid authentication format", http.StatusUnauthorized)
return
}

// 验证令牌
token := parts[1]
if !validator(token) {
http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
return
}

// 认证通过，继续处理
next.ServeHTTP(w, r)
})
}
}

// RateLimitMiddleware 创建速率限制中间件
func RateLimitMiddleware(limit int, window time.Duration) interfaces.Middleware {
// 简单实现，实际项目中可以使用更复杂的限流算法
type client struct {
count    int
lastSeen time.Time
}

clients := make(map[string]*client)

return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 获取客户端IP
ip := r.RemoteAddr

// 检查并更新计数
now := time.Now()
c, exists := clients[ip]

if !exists {
clients[ip] = &client{count: 1, lastSeen: now}
} else {
// 如果超过窗口期，重置计数
if now.Sub(c.lastSeen) > window {
c.count = 1
c.lastSeen = now
} else {
// 增加计数
c.count++
c.lastSeen = now

// 检查是否超过限制
if c.count > limit {
http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
return
}
}
}

// 未超过限制，继续处理
next.ServeHTTP(w, r)
})
}
}

// TimeoutMiddleware 创建超时中间件
func TimeoutMiddleware(timeout time.Duration) interfaces.Middleware {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 创建带超时的上下文
ctx, cancel := context.WithTimeout(r.Context(), timeout)
defer cancel()

// 使用新上下文
r = r.WithContext(ctx)

// 处理请求
done := make(chan bool)
go func() {
next.ServeHTTP(w, r)
done <- true
}()

// 等待处理完成或超时
select {
case <-done:
// 请求正常完成
return
case <-ctx.Done():
// 请求超时
http.Error(w, "Request Timeout", http.StatusGatewayTimeout)
return
}
})
}
}

// TracingMiddleware 创建追踪中间件
func TracingMiddleware(tracer interfaces.Tracer) interfaces.Middleware {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 创建追踪上下文
// 开始追踪
		traceID := tracer.StartTrace("api_request")
		spanID := tracer.StartSpan(traceID, "http_request", "")
		
		// 添加标签
		tracer.AddTag(traceID, spanID, "method", r.Method)
		tracer.AddTag(traceID, spanID, "path", r.URL.Path)
		tracer.AddTag(traceID, spanID, "user_agent", r.UserAgent())
		
		// 处理请求
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		
		// 添加响应信息
		tracer.AddTag(traceID, spanID, "status_code", string(rune(rw.statusCode)))
		tracer.AddMetric(traceID, spanID, "duration_ms", float64(duration.Nanoseconds())/1e6)
		tracer.EndSpan(traceID, spanID)
})
}
}

// responseWriter 包装http.ResponseWriter以捕获状态码
type responseWriter struct {
http.ResponseWriter
statusCode int
}

// WriteHeader 重写以捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
rw.statusCode = code
rw.ResponseWriter.WriteHeader(code)
}
