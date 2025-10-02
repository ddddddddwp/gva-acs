package router

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/api"
)

type OperationLogRouter struct{}

// InitOperationLogRouter 初始化操作日志路由
func (o *OperationLogRouter) InitOperationLogRouter(Router *gin.RouterGroup) {
	operationLogRouter := Router.Group("operationLog").Use(middleware.OperationRecord())
	operationLogRouterWithoutRecord := Router.Group("operationLog")
	operationLogApi := api.ApiGroupApp.OperationLogApi

	{
		// 需要记录操作的路由
		operationLogRouter.PUT("updateOperationLog", operationLogApi.UpdateOperationLog)                       // 更新操作日志
		operationLogRouter.PUT("completeOperationLog", operationLogApi.CompleteOperationLog)                   // 完成操作日志
		operationLogRouter.POST("retryFailedOperation/:id", operationLogApi.RetryFailedOperation)              // 重试失败操作
		operationLogRouter.POST("cleanupOldOperationLogs", operationLogApi.CleanupOldOperationLogs)            // 清理旧操作日志
		operationLogRouter.DELETE("deleteOperationLog/:id", operationLogApi.DeleteOperationLog)                // 删除操作日志
	}

	{
		// 不需要记录操作的路由
		operationLogRouterWithoutRecord.POST("getOperationLogList", operationLogApi.GetOperationLogList)                           // 获取操作日志列表
		operationLogRouterWithoutRecord.GET("getOperationLogByID/:id", operationLogApi.GetOperationLogByID)                        // 根据ID获取操作日志
		operationLogRouterWithoutRecord.POST("getOperationLogsByDevice/:deviceId", operationLogApi.GetOperationLogsByDevice)       // 获取设备操作日志
		operationLogRouterWithoutRecord.POST("getOperationLogsBySession/:sessionId", operationLogApi.GetOperationLogsBySession)    // 获取会话操作日志
		operationLogRouterWithoutRecord.POST("getOperationLogStatistics", operationLogApi.GetOperationLogStatistics)              // 获取操作日志统计
		operationLogRouterWithoutRecord.GET("getRecentOperationLogs", operationLogApi.GetRecentOperationLogs)                      // 获取最近操作日志
		operationLogRouterWithoutRecord.POST("getFailedOperationLogs", operationLogApi.GetFailedOperationLogs)                     // 获取失败操作日志
	}
}