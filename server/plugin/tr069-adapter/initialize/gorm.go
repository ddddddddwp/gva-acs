package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
)

// InitializeDB 初始化数据库
func InitializeDB() {
	// 自动迁移TR069设备和事件表
	if err := global.GVA_DB.AutoMigrate(
		model.TR069Device{},
		model.TR069Event{},
	); err != nil {
		global.GVA_LOG.Error("TR069-Adapter插件表结构迁移失败!")
	} else {
		global.GVA_LOG.Info("TR069-Adapter插件表结构迁移成功!")
	}
}