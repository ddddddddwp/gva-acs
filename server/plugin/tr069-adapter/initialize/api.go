package initialize

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const initOrderApi = 20

type InitializerApi struct{}

// auto run
func init() {
	RegisterInit(initOrderApi, &InitializerApi{})
}

func (i *InitializerApi) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, nil
	}
	return ctx, i.initializeApi(db)
}

func (i *InitializerApi) InitializerName() string {
	return "tr069-adapter.api"
}

func (i *InitializerApi) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	
	// 检查是否已经初始化过API
	var count int64
	db.Model(&sysModel.SysApi{}).Where("api_group = ?", "TR069-Adapter").Count(&count)
	return count > 0
}

// initializeApi 初始化API
func (i *InitializerApi) initializeApi(db *gorm.DB) error {
	global.GVA_LOG.Info("开始初始化TR069-Adapter插件API")

	apis := []sysModel.SysApi{
		// 设备管理API
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/device/getDeviceList", Description: "获取设备列表"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/device/getDeviceByID/:id", Description: "根据ID获取设备"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/device/getDeviceBySerialNumber/:sn", Description: "根据序列号获取设备"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/device/createDevice", Description: "创建设备"},
		{ApiGroup: "TR069-Adapter", Method: "PUT", Path: "/tr069-adapter/device/updateDevice", Description: "更新设备"},
		{ApiGroup: "TR069-Adapter", Method: "DELETE", Path: "/tr069-adapter/device/deleteDevice/:id", Description: "删除设备"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/device/batchDeleteDevices", Description: "批量删除设备"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/device/getDeviceStats", Description: "获取设备统计"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/device/rebootDevice/:id", Description: "重启设备"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/device/factoryResetDevice/:id", Description: "恢复出厂设置"},

		// 参数管理API
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/getParameterList", Description: "获取参数列表"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/parameter/getParametersByDevice/:deviceId", Description: "获取设备参数"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/parameter/getParameterTree/:deviceId", Description: "获取参数树"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/getParameterValues", Description: "获取参数值"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/setParameterValues", Description: "设置参数值"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/getParameterNames", Description: "获取参数名称"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/getParameterAttributes", Description: "获取参数属性"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/setParameterAttributes", Description: "设置参数属性"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/addObject", Description: "添加对象"},
		{ApiGroup: "TR069-Adapter", Method: "DELETE", Path: "/tr069-adapter/parameter/deleteObject", Description: "删除对象"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/parameter/searchParameters", Description: "搜索参数"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/parameter/getParameterStatistics", Description: "获取参数统计"},

		// 会话管理API
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/session/getSessionList", Description: "获取会话列表"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/session/getSessionByID/:id", Description: "根据ID获取会话"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/session/getActiveSessionsByDevice/:deviceId", Description: "获取设备活跃会话"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/session/endSession/:id", Description: "结束会话"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/session/getDeviceSessionHistory/:deviceId", Description: "获取设备会话历史"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/session/getSessionStatistics", Description: "获取会话统计"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/session/cleanupExpiredSessions", Description: "清理过期会话"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/session/getSessionsByTimeRange", Description: "按时间范围获取会话"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/session/isSessionActive/:sessionId", Description: "检查会话是否活跃"},

		// 操作日志API
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/getOperationLogList", Description: "获取操作日志列表"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/operationLog/getOperationLogByID/:id", Description: "根据ID获取操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/getOperationLogsByDevice/:deviceId", Description: "获取设备操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/getOperationLogsBySession/:sessionId", Description: "获取会话操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/getOperationLogStatistics", Description: "获取操作日志统计"},
		{ApiGroup: "TR069-Adapter", Method: "GET", Path: "/tr069-adapter/operationLog/getRecentOperationLogs", Description: "获取最近操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/getFailedOperationLogs", Description: "获取失败操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/retryFailedOperation/:id", Description: "重试失败操作"},
		{ApiGroup: "TR069-Adapter", Method: "POST", Path: "/tr069-adapter/operationLog/cleanupOldOperationLogs", Description: "清理旧操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "PUT", Path: "/tr069-adapter/operationLog/updateOperationLog", Description: "更新操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "PUT", Path: "/tr069-adapter/operationLog/completeOperationLog", Description: "完成操作日志"},
		{ApiGroup: "TR069-Adapter", Method: "DELETE", Path: "/tr069-adapter/operationLog/deleteOperationLog/:id", Description: "删除操作日志"},
	}

	// 批量创建API
	for _, api := range apis {
		var existingApi sysModel.SysApi
		err := db.Where("path = ? AND method = ?", api.Path, api.Method).First(&existingApi).Error
		if err == gorm.ErrRecordNotFound {
			// API不存在，创建新的
			if err := db.Create(&api).Error; err != nil {
				global.GVA_LOG.Error("创建API失败", zap.String("path", api.Path), zap.Error(err))
				return err
			}
		}
	}

	global.GVA_LOG.Info("TR069-Adapter插件API初始化完成")
	return nil
}