package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// SessionSearch 会话搜索请求
type SessionSearch struct {
	request.PageInfo
	DeviceID  uint   `json:"deviceId" form:"deviceId"`
	Status    string `json:"status" form:"status"`
	StartTime string `json:"startTime" form:"startTime"`
	EndTime   string `json:"endTime" form:"endTime"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	DeviceID uint   `json:"deviceId" binding:"required"`
	Type     string `json:"type" binding:"required"`
}

// CloseSessionRequest 关闭会话请求
type CloseSessionRequest struct {
	SessionID uint `json:"sessionId" binding:"required"`
}

// SessionByID 根据ID获取会话请求
type SessionByID struct {
	ID uint `json:"id" form:"id" binding:"required"`
}