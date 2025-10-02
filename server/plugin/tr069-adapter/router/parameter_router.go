package router

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/api"
)

type ParameterRouter struct{}

// InitParameterRouter 初始化参数路由
func (p *ParameterRouter) InitParameterRouter(Router *gin.RouterGroup) {
	parameterRouter := Router.Group("parameter").Use(middleware.OperationRecord())
	parameterRouterWithoutRecord := Router.Group("parameter")
	parameterApi := api.ApiGroupApp.ParameterApi

	{
		// 需要记录操作的路由
		parameterRouter.POST("setParameterValues", parameterApi.SetParameterValues)           // 设置参数值
		parameterRouter.POST("setParameterAttributes", parameterApi.SetParameterAttributes)   // 设置参数属性
		parameterRouter.POST("addObject", parameterApi.AddObject)                             // 添加对象
		parameterRouter.DELETE("deleteObject", parameterApi.DeleteObject)                     // 删除对象
	}

	{
		// 不需要记录操作的路由
		parameterRouterWithoutRecord.POST("getParameterList", parameterApi.GetParameterList)                     // 获取参数列表
		parameterRouterWithoutRecord.GET("getParametersByDevice/:deviceId", parameterApi.GetParametersByDevice)  // 获取设备参数
		parameterRouterWithoutRecord.GET("getParameterTree/:deviceId", parameterApi.GetParameterTree)            // 获取参数树
		parameterRouterWithoutRecord.POST("getParameterValues", parameterApi.GetParameterValues)                 // 获取参数值
		parameterRouterWithoutRecord.POST("getParameterNames", parameterApi.GetParameterNames)                   // 获取参数名称
		parameterRouterWithoutRecord.POST("getParameterAttributes", parameterApi.GetParameterAttributes)         // 获取参数属性
		parameterRouterWithoutRecord.POST("searchParameters", parameterApi.SearchParameters)                     // 搜索参数
		parameterRouterWithoutRecord.GET("getParameterStatistics", parameterApi.GetParameterStatistics)          // 获取参数统计
	}
}