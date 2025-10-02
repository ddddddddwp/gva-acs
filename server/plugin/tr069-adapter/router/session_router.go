package router

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/api"
)

type SessionRouter struct{}

// InitSessionRouter 初始化会话路由
func (s *SessionRouter) InitSessionRouter(Router *gin.RouterGroup) {
	sessionRouter := Router.Group("session").Use(middleware.OperationRecord())
	sessionRouterWithoutRecord := Router.Group("session")
	sessionApi := api.ApiGroupApp.SessionApi

	{
		// 需要记录操作的路由
		sessionRouter.POST("endSession/:id", sessionApi.EndSession)                         // 结束会话
		sessionRouter.POST("cleanupExpiredSessions", sessionApi.CleanupExpiredSessions)     // 清理过期会话
	}

	{
		// 不需要记录操作的路由
		sessionRouterWithoutRecord.POST("getSessionList", sessionApi.GetSessionList)                           // 获取会话列表
		sessionRouterWithoutRecord.GET("getSessionByID/:id", sessionApi.GetSessionByID)                        // 根据ID获取会话
		sessionRouterWithoutRecord.GET("getActiveSessionsByDevice/:deviceId", sessionApi.GetActiveSessionsByDevice) // 获取设备活跃会话
		sessionRouterWithoutRecord.GET("getDeviceSessionHistory/:deviceId", sessionApi.GetDeviceSessionHistory)     // 获取设备会话历史
		sessionRouterWithoutRecord.POST("getSessionStatistics", sessionApi.GetSessionStatistics)               // 获取会话统计
		sessionRouterWithoutRecord.POST("getSessionsByTimeRange", sessionApi.GetSessionsByTimeRange)           // 按时间范围获取会话
		sessionRouterWithoutRecord.GET("isSessionActive/:sessionId", sessionApi.IsSessionActive)               // 检查会话是否活跃
	}
}