package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
)

// InitializeDB 初始化数据库
func InitializeDB() error {
	// 自动迁移表结构
	return global.GVA_DB.AutoMigrate(
		model.TR069Device{},
		model.Parameter{},
	)
}