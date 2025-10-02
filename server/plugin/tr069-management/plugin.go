package tr069management

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/initialize"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

type ManagementPlugin struct{}

// Register 注册TR069-Management插件
func (p *ManagementPlugin) Register(group *gin.Engine) {
	// 初始化菜单
	if err := initialize.InitializeMenu(); err != nil {
		global.GVA_LOG.Error("TR069-Management插件菜单初始化失败", zap.Error(err))
	} else {
		global.GVA_LOG.Info("TR069-Management插件菜单初始化成功")
	}
	
	// 创建路由组
	routerGroup := group.Group(p.RouterPath())
	initialize.InitializeRouter(routerGroup)
}

// RouterPath 定义插件路由路径
func (p *ManagementPlugin) RouterPath() string {
	return "tr069-management"
}