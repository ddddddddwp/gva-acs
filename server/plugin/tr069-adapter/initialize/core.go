package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	adapterGlobal "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
)

// InitializeTR069Core 初始化TR069-core组件
func InitializeTR069Core() error {
	// 初始化解析器
	parser := factory.NewParser()
	adapterGlobal.TR069Parser = parser

	// 初始化构建器
	builder := factory.NewBuilder()
	adapterGlobal.TR069Builder = builder

	// 初始化事件管理器
	eventManager := factory.CreateEventNotifier()
	adapterGlobal.TR069EventManager = eventManager

	global.GVA_LOG.Info("TR069-core组件初始化成功")
	return nil
}