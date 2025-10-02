package initialize

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const initOrderMenu = 30

type InitializerMenu struct{}

// auto run
func init() {
	RegisterInit(initOrderMenu, &InitializerMenu{})
}

func (i *InitializerMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, nil
	}
	return ctx, i.initializeMenu(db)
}

func (i *InitializerMenu) InitializerName() string {
	return "tr069-adapter.menu"
}

func (i *InitializerMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	
	// 检查是否已经初始化过菜单
	var count int64
	db.Model(&sysModel.SysBaseMenu{}).Where("name = ?", "tr069Adapter").Count(&count)
	return count > 0
}

// initializeMenu 初始化菜单
func (i *InitializerMenu) initializeMenu(db *gorm.DB) error {
	global.GVA_LOG.Info("开始初始化TR069-Adapter插件菜单")

	// 创建主菜单
	mainMenu := sysModel.SysBaseMenu{
		MenuLevel: 0,
		ParentId:  0,
		Path:      "tr069Adapter",
		Name:      "tr069Adapter",
		Hidden:    false,
		Component: "view/routerHolder.vue",
		Sort:      100,
		Meta: sysModel.Meta{
			Title: "TR069适配器",
			Icon:  "router",
		},
	}

	// 检查主菜单是否存在
	var existingMainMenu sysModel.SysBaseMenu
	err := db.Where("name = ?", mainMenu.Name).First(&existingMainMenu).Error
	if err == gorm.ErrRecordNotFound {
		// 主菜单不存在，创建
		if err := db.Create(&mainMenu).Error; err != nil {
			global.GVA_LOG.Error("创建TR069-Adapter主菜单失败", zap.Error(err))
			return err
		}
		existingMainMenu = mainMenu
	}

	// 子菜单列表
	subMenus := []sysModel.SysBaseMenu{
		{
			MenuLevel: 0,
			ParentId:  existingMainMenu.ID,
			Path:      "deviceManagement",
			Name:      "deviceManagement",
			Hidden:    false,
			Component: "plugin/tr069-adapter/view/device/index.vue",
			Sort:      1,
			Meta: sysModel.Meta{
				Title:     "设备管理",
				Icon:      "monitor",
				KeepAlive: true,
			},
		},
		{
			MenuLevel: 0,
			ParentId:  existingMainMenu.ID,
			Path:      "parameterManagement",
			Name:      "parameterManagement",
			Hidden:    false,
			Component: "plugin/tr069-adapter/view/parameter/index.vue",
			Sort:      2,
			Meta: sysModel.Meta{
				Title:     "参数管理",
				Icon:      "setting",
				KeepAlive: true,
			},
		},
		{
			MenuLevel: 0,
			ParentId:  existingMainMenu.ID,
			Path:      "sessionManagement",
			Name:      "sessionManagement",
			Hidden:    false,
			Component: "plugin/tr069-adapter/view/session/index.vue",
			Sort:      3,
			Meta: sysModel.Meta{
				Title:     "会话管理",
				Icon:      "connection",
				KeepAlive: true,
			},
		},
		{
			MenuLevel: 0,
			ParentId:  existingMainMenu.ID,
			Path:      "operationLogManagement",
			Name:      "operationLogManagement",
			Hidden:    false,
			Component: "plugin/tr069-adapter/view/operationLog/index.vue",
			Sort:      4,
			Meta: sysModel.Meta{
				Title:     "操作日志",
				Icon:      "document",
				KeepAlive: true,
			},
		},
		{
			MenuLevel: 0,
			ParentId:  existingMainMenu.ID,
			Path:      "tr069Dashboard",
			Name:      "tr069Dashboard",
			Hidden:    false,
			Component: "plugin/tr069-adapter/view/dashboard/index.vue",
			Sort:      5,
			Meta: sysModel.Meta{
				Title:     "监控面板",
				Icon:      "dashboard",
				KeepAlive: true,
			},
		},
	}

	// 创建子菜单
	for _, menu := range subMenus {
		var existingMenu sysModel.SysBaseMenu
		err := db.Where("name = ? AND parent_id = ?", menu.Name, menu.ParentId).First(&existingMenu).Error
		if err == gorm.ErrRecordNotFound {
			// 菜单不存在，创建新的
			if err := db.Create(&menu).Error; err != nil {
				global.GVA_LOG.Error("创建子菜单失败", zap.String("name", menu.Name), zap.Error(err))
				return err
			}
		}
	}

	global.GVA_LOG.Info("TR069-Adapter插件菜单初始化完成")
	return nil
}