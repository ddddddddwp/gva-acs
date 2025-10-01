package initialize

import (
	"github.com/gin-gonic/gin"
)

// InitializeRouter 初始化路由
func InitializeRouter(group *gin.RouterGroup) {
	// TR-069插件不需要额外的HTTP路由，因为它使用独立的7547端口服务器
}