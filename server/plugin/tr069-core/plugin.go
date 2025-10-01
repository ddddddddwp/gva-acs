package tr069

import (
	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069/initialize"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069/server"
)

type tr069Plugin struct{}

// Register 注册插件
func (e *tr069Plugin) Register(group *gin.RouterGroup) {
	initialize.InitializeRouter(group)
	// 启动TR-069服务器
	go server.StartTR069Server()
}

// RouterPath 返回插件路由路径
func (e *tr069Plugin) RouterPath() string {
	return "tr069"
}

func CreateTr069Plugin() *tr069Plugin {
	return &tr069Plugin{}
}