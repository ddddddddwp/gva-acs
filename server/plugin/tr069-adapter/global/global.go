package global

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/config"
	"sync"
)

var (
	// TR069Config 插件配置
	TR069Config config.TR069AdapterConfig

	// DeviceCache 设备缓存
	DeviceCache sync.Map
)

// InitGlobalVariables 初始化全局变量
func InitGlobalVariables() {
	// 初始化设备缓存
	DeviceCache = sync.Map{}
}