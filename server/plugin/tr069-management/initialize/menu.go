package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// InitializeMenu 初始化菜单
func InitializeMenu() error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var parentMenu system.SysBaseMenu
		// 查找或创建父菜单
		if err := tx.Where("name = ?", "tr069Management").First(&parentMenu).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建父菜单
			parentMenu = system.SysBaseMenu{
				MenuLevel: 0,
				ParentId:  0,
				Path:      "tr069Management",
				Name:      "tr069Management",
				Hidden:    false,
				Component: "view/tr069Management/index.vue",
				Sort:      10,
				Meta: system.Meta{
					Title: "TR069设备管理",
					Icon:  "monitor",
				},
			}
			if err := tx.Create(&parentMenu).Error; err != nil {
				return err
			}
		}

		// 子菜单列表
		childMenus := []system.SysBaseMenu{
			{
				MenuLevel: 1,
				ParentId:  parentMenu.ID,
				Path:      "deviceList",
				Name:      "deviceList",
				Hidden:    false,
				Component: "view/tr069Management/device/index.vue",
				Sort:      1,
				Meta: system.Meta{
					Title: "设备列表",
					Icon:  "cpu",
				},
			},
			{
				MenuLevel: 1,
				ParentId:  parentMenu.ID,
				Path:      "parameterManage",
				Name:      "parameterManage",
				Hidden:    false,
				Component: "view/tr069Management/parameter/index.vue",
				Sort:      2,
				Meta: system.Meta{
					Title: "参数管理",
					Icon:  "setting",
				},
			},
			{
				MenuLevel: 1,
				ParentId:  parentMenu.ID,
				Path:      "groupManage",
				Name:      "groupManage",
				Hidden:    false,
				Component: "view/tr069Management/group/index.vue",
				Sort:      3,
				Meta: system.Meta{
					Title: "设备分组",
					Icon:  "folder",
				},
			},
			{
				MenuLevel: 1,
				ParentId:  parentMenu.ID,
				Path:      "configManage",
				Name:      "configManage",
				Hidden:    false,
				Component: "view/tr069Management/config/index.vue",
				Sort:      4,
				Meta: system.Meta{
					Title: "配置管理",
					Icon:  "files",
				},
			},
			{
				MenuLevel: 1,
				ParentId:  parentMenu.ID,
				Path:      "firmwareManage",
				Name:      "firmwareManage",
				Hidden:    false,
				Component: "view/tr069Management/firmware/index.vue",
				Sort:      5,
				Meta: system.Meta{
					Title: "固件管理",
					Icon:  "upload",
				},
			},
		}

		// 创建子菜单
		for i := range childMenus {
			var menu system.SysBaseMenu
			if err := tx.Where("name = ?", childMenus[i].Name).First(&menu).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&childMenus[i]).Error; err != nil {
					return err
				}
			}
		}

		// 创建API权限
		apis := []system.SysApi{
			{Path: "/tr069-management/device/list", Description: "获取设备列表", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/device/detail/:id", Description: "获取设备详情", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/device/create", Description: "创建设备", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/device/update", Description: "更新设备", ApiGroup: "TR069设备管理", Method: "PUT"},
			{Path: "/tr069-management/device/delete/:id", Description: "删除设备", ApiGroup: "TR069设备管理", Method: "DELETE"},
			{Path: "/tr069-management/device/batchDelete", Description: "批量删除设备", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/device/stats", Description: "获取设备统计信息", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/device/status", Description: "更新设备状态", ApiGroup: "TR069设备管理", Method: "PUT"},
			
			{Path: "/tr069-management/parameter/list", Description: "获取设备参数列表", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/parameter/detail/:id", Description: "获取参数详情", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/parameter/set", Description: "设置设备参数", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/parameter/categories", Description: "获取参数分类列表", ApiGroup: "TR069设备管理", Method: "GET"},
			
			{Path: "/tr069-management/group/list", Description: "获取设备组列表", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/group/detail/:id", Description: "获取设备组详情", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/group/create", Description: "创建设备组", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/group/update", Description: "更新设备组", ApiGroup: "TR069设备管理", Method: "PUT"},
			{Path: "/tr069-management/group/delete/:id", Description: "删除设备组", ApiGroup: "TR069设备管理", Method: "DELETE"},
			{Path: "/tr069-management/group/addDevice", Description: "添加设备到设备组", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/group/removeDevice", Description: "从设备组移除设备", ApiGroup: "TR069设备管理", Method: "DELETE"},
			
			{Path: "/tr069-management/config/list", Description: "获取配置文件列表", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/config/detail/:id", Description: "获取配置文件详情", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/config/create", Description: "创建配置文件", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/config/update", Description: "更新配置文件", ApiGroup: "TR069设备管理", Method: "PUT"},
			{Path: "/tr069-management/config/delete/:id", Description: "删除配置文件", ApiGroup: "TR069设备管理", Method: "DELETE"},
			{Path: "/tr069-management/config/apply", Description: "应用配置文件到设备", ApiGroup: "TR069设备管理", Method: "POST"},
			
			{Path: "/tr069-management/firmware/list", Description: "获取固件列表", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/firmware/detail/:id", Description: "获取固件详情", ApiGroup: "TR069设备管理", Method: "GET"},
			{Path: "/tr069-management/firmware/create", Description: "创建固件", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/firmware/update", Description: "更新固件", ApiGroup: "TR069设备管理", Method: "PUT"},
			{Path: "/tr069-management/firmware/delete/:id", Description: "删除固件", ApiGroup: "TR069设备管理", Method: "DELETE"},
			{Path: "/tr069-management/firmware/upload", Description: "上传固件", ApiGroup: "TR069设备管理", Method: "POST"},
			{Path: "/tr069-management/firmware/upgrade", Description: "升级设备固件", ApiGroup: "TR069设备管理", Method: "POST"},
		}

		// 创建API
		for i := range apis {
			var api system.SysApi
			if err := tx.Where("path = ? AND method = ?", apis[i].Path, apis[i].Method).First(&api).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&apis[i]).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}