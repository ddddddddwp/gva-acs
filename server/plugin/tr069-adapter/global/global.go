package global

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/config"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

var (
	// TR069Config TR069适配器配置
	TR069Config *config.TR069AdapterConfig

	// DeviceCache 设备缓存
	DeviceCache map[string]interface{}

	// TR069-core接口实例
	TR069Parser       interfaces.Parser
	TR069Builder      interfaces.Builder
	TR069EventManager interfaces.EventNotifier
)

// InitGlobalVariables 初始化全局变量
func InitGlobalVariables() {
	DeviceCache = make(map[string]interface{})
}
