package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type ParameterRouter struct{}

// InitParameterRouter 初始化参数路由
func (r *ParameterRouter) InitParameterRouter(Router *gin.RouterGroup) {
	parameterRouter := Router.Group("parameter").Use(middleware.OperationRecord())
	parameterRouterWithoutRecord := Router.Group("parameter")
	
	parameterApi := apiGroupApp.ParameterApi
	{
		parameterRouter.POST("set", parameterApi.SetDeviceParameters) // 设置设备参数
	}
	{
		parameterRouterWithoutRecord.GET("list", parameterApi.GetParametersByDevice)   // 获取设备参数列表
		parameterRouterWithoutRecord.GET("detail/:id", parameterApi.GetParameterByID)  // 获取参数详情
		parameterRouterWithoutRecord.GET("categories", parameterApi.GetParameterCategories) // 获取参数分类列表
	}
}