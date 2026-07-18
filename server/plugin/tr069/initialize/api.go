package initialize

import (
	"context"

	model "github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/ddddddddwp/gva-acs/server/plugin/plugin-tool/utils"
)

func Api(ctx context.Context) {
	entities := []model.SysApi{
		// Device APIs
		{Path: "/tr069/device/list", Description: "获取设备列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/device", Description: "创建设备", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/device/:deviceId", Description: "删除设备", ApiGroup: "TR069", Method: "DELETE"},
		{Path: "/tr069/device/:deviceId/connection-profile", Description: "获取设备 Connection Profile", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/device/:deviceId/connection-profile", Description: "更新设备 Connection Profile", ApiGroup: "TR069", Method: "PUT"},

		// FAP APIs
		{Path: "/tr069/fap/:deviceId", Description: "获取FAP信息", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/fap/:deviceId/sync", Description: "同步FAP信息", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/fap/:deviceId", Description: "配置FAP", ApiGroup: "TR069", Method: "PUT"},

		// Debug APIs
		{Path: "/tr069/debug/trace/:traceId", Description: "获取调试追踪", ApiGroup: "TR069", Method: "GET"},

		// Command APIs
		{Path: "/tr069/command/:deviceId/getRPCMethods", Description: "查询设备能力", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/getParameterValues", Description: "获取参数值", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/getParameterNames", Description: "获取参数名称", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/getParameterAttributes", Description: "获取参数属性", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/setParameterValues", Description: "设置参数值", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/setParameterAttributes", Description: "设置参数属性", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/addObject", Description: "添加CWMP对象", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/deleteObject", Description: "删除CWMP对象", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/download", Description: "下发文件下载", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/upload", Description: "下发文件上传", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/reboot", Description: "重启设备", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/command/:deviceId/factoryReset", Description: "恢复出厂设置", ApiGroup: "TR069", Method: "POST"},

		// Command record APIs
		{Path: "/tr069/command-record/list", Description: "RPC记录列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/command-record/:commandId", Description: "RPC记录详情", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/command-record/:commandId/retry", Description: "RPC记录重新下发", ApiGroup: "TR069", Method: "POST"},

		// Log artifact APIs
		{Path: "/tr069/artifact/list", Description: "获取基站日志文件列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/artifact/:artifactId/download", Description: "下载基站日志文件", ApiGroup: "TR069", Method: "GET"},

		// DataModel APIs
		{Path: "/tr069/datamodel/:deviceId/sync", Description: "全量同步数据模型", ApiGroup: "TR069", Method: "POST"},
		{Path: "/tr069/datamodel/:deviceId/list", Description: "获取数据模型列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/datamodel/:deviceId/structure", Description: "获取数据模型结构", ApiGroup: "TR069", Method: "GET"},

		// Alarm APIs
		{Path: "/tr069/alarm/list", Description: "获取当前告警列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/alarm/history", Description: "获取历史告警列表", ApiGroup: "TR069", Method: "GET"},
		{Path: "/tr069/alarm/stats", Description: "获取告警统计", ApiGroup: "TR069", Method: "GET"},
	}
	utils.RegisterApis(entities...)
}
