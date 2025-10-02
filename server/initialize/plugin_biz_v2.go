package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/announcement"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/plugin/v2"
	"github.com/gin-gonic/gin"
	
	// tr069-adapter 插件
	tr069adapter "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter"
	// tr069-management 插件
	tr069management "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management"
)

func PluginInitV2(group *gin.Engine, plugins ...plugin.Plugin) {
	for i := 0; i < len(plugins); i++ {
		plugins[i].Register(group)
	}
}
func bizPluginV2(engine *gin.Engine) {
	// 恢复tr069-adapter插件的初始化
	// 添加tr069-management插件的初始化
	PluginInitV2(engine, announcement.Plugin, tr069adapter.NewPlugin(), &tr069management.ManagementPlugin{})
}
