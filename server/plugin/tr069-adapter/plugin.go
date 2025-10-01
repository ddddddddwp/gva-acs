package tr069_adapter

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/initialize"
	interfaces "github.com/flipped-aurora/gin-vue-admin/server/utils/plugin/v2"
	"github.com/gin-gonic/gin"
)

var _ interfaces.Plugin = (*tr069AdapterPlugin)(nil)

type tr069AdapterPlugin struct{}

// Register 注册插件
func (p *tr069AdapterPlugin) Register(group *gin.Engine) {
	// 初始化配置
	initialize.InitializeViper()
	
	// 初始化数据库
	initialize.InitializeDB()
	
	// 创建路由组
	routerGroup := group.Group("tr069-adapter")
	
	// 初始化路由
	initialize.InitializeRouter(routerGroup)
	
	// 初始化TR069服务器
	initialize.InitializeTR069Server()
	
	global.GVA_LOG.Info("TR069-Adapter插件注册成功!")
}

// NewPlugin 创建插件实例
func NewPlugin() *tr069AdapterPlugin {
	return &tr069AdapterPlugin{}
}