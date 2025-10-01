package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/router"
	"github.com/gin-gonic/gin"
)

// InitializeRouter 初始化路由
func InitializeRouter(Router *gin.RouterGroup) {
	router.RouterGroupApp.TR069Router.InitTR069Router(Router)
}