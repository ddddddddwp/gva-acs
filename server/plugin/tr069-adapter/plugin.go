package tr069_adapter

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/initialize"
	"github.com/gin-gonic/gin"
)

type TR069AdapterPlugin struct{}

// Register 注册插件
func (p *TR069AdapterPlugin) Register(engine *gin.Engine) {
	// 创建插件路由组
	group := engine.Group(p.RouterPath())
	// 初始化路由
	initialize.InitializeRouter(group)
}

// RouterPath 返回插件路由路径
func (p *TR069AdapterPlugin) RouterPath() string {
	return "tr069-adapter"
}

// PluginName 返回插件名称
func (p *TR069AdapterPlugin) PluginName() string {
	return "TR069-Adapter"
}

// Description 返回插件描述
func (p *TR069AdapterPlugin) Description() string {
	return "TR069协议适配器插件，提供CPE设备管理、参数配置、会话管理和操作日志功能"
}

// Version 返回插件版本
func (p *TR069AdapterPlugin) Version() string {
	return "v1.0.0"
}

// Author 返回插件作者
func (p *TR069AdapterPlugin) Author() string {
	return "GVA Team"
}

// NewPlugin 创建插件实例
func NewPlugin() *TR069AdapterPlugin {
	return &TR069AdapterPlugin{}
}

// CreatePlugin 创建插件实例（兼容性函数）
func CreatePlugin() *TR069AdapterPlugin {
	return &TR069AdapterPlugin{}
}