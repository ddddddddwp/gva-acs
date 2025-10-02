package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/api/rest"
)

// SessionHandler 处理TR069会话相关的API请求
type SessionHandler struct {
	sessionManager interfaces.SessionManager
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(sessionManager interfaces.SessionManager) *SessionHandler {
	return &SessionHandler{
		sessionManager: sessionManager,
	}
}

// RegisterRoutes 注册API路由
func (h *SessionHandler) RegisterRoutes(router interfaces.Router) {
	router.Handle(http.MethodGet, "/sessions", h.ListSessions)
	router.Handle(http.MethodGet, "/sessions/{id}", h.GetSession)
	router.Handle(http.MethodDelete, "/sessions/{id}", h.CloseSession)
	router.Handle(http.MethodPost, "/sessions", h.CreateSession)
}

// SetMiddleware 设置中间件
func (h *SessionHandler) SetMiddleware(middleware ...interfaces.Middleware) {
	// 这里可以存储特定于处理器的中间件
}

// ListSessions 列出所有会话
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	// 在实际应用中，这里会从会话管理器获取会话列表
	// 模拟会话列表
	sessions := []map[string]interface{}{
		{
			"id":        "session1",
			"deviceId":  "device1",
			"startTime": "2023-05-01T10:00:00Z",
			"status":    "active",
		},
		{
			"id":        "session2",
			"deviceId":  "device2",
			"startTime": "2023-05-01T11:00:00Z",
			"status":    "closed",
		},
	}

	rest.SendSuccessResponse(w, sessions)
}

// GetSession 获取单个会话信息
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取会话ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid session ID", 400)
		return
	}

	// 在实际应用中，这里会从会话管理器获取会话信息
	// 模拟会话信息
	session := map[string]interface{}{
		"id":            id,
		"deviceId":      "device1",
		"startTime":     "2023-05-01T10:00:00Z",
		"lastActivity":  "2023-05-01T10:15:00Z",
		"status":        "active",
		"messageCount":  5,
		"clientAddress": "192.168.1.100",
	}

	rest.SendSuccessResponse(w, session)
}

// CloseSession 关闭会话
func (h *SessionHandler) CloseSession(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取会话ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid session ID", 400)
		return
	}

	// 在实际应用中，这里会调用会话管理器关闭会话
	// 模拟关闭会话
	response := map[string]interface{}{
		"id":      id,
		"status":  "closed",
		"message": "Session closed successfully",
	}

	rest.SendSuccessResponse(w, response)
}

// CreateSession 创建新会话
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var sessionRequest struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&sessionRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}

	if sessionRequest.DeviceID == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Device ID is required", 400)
		return
	}

	// 在实际应用中，这里会调用会话管理器创建会话
	// 模拟创建会话
	session := map[string]interface{}{
		"id":            "session-new",
		"deviceId":      sessionRequest.DeviceID,
		"startTime":     "2023-05-01T12:00:00Z",
		"lastActivity":  "2023-05-01T12:00:00Z",
		"status":        "active",
		"messageCount":  0,
		"clientAddress": "192.168.1.100",
	}

	rest.SendSuccessResponse(w, session)
}