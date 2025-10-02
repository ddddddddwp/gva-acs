package initialize

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const initOrderTable = 10

type InitializerTable struct{}

// auto run
func init() {
	RegisterInit(initOrderTable, &InitializerTable{})
}

func (i *InitializerTable) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, nil
	}
	return ctx, i.initializeTable(db)
}

func (i *InitializerTable) InitializerName() string {
	return "tr069-adapter.table"
}

func (i *InitializerTable) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	
	// 检查是否已经创建了表
	return db.Migrator().HasTable(&model.CpeDevice{}) &&
		db.Migrator().HasTable(&model.CpeParameter{}) &&
		db.Migrator().HasTable(&model.CpeSession{}) &&
		db.Migrator().HasTable(&model.CpeOperationLog{})
}

// initializeTable 初始化数据库表
func (i *InitializerTable) initializeTable(db *gorm.DB) error {
	global.GVA_LOG.Info("开始初始化TR069-Adapter插件数据库表")

	// 自动迁移数据库表
	err := db.AutoMigrate(
		&model.CpeDevice{},
		&model.CpeParameter{},
		&model.CpeSession{},
		&model.CpeOperationLog{},
	)

	if err != nil {
		global.GVA_LOG.Error("TR069-Adapter插件数据库表迁移失败", zap.Error(err))
		return err
	}

	global.GVA_LOG.Info("TR069-Adapter插件数据库表迁移完成")
	return nil
}