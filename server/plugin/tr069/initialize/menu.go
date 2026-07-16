package initialize

import (
	"context"
	"github.com/ddddddddwp/gva-acs/server/global"
	model "github.com/ddddddddwp/gva-acs/server/model/system"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Menu(ctx context.Context) {
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 一级菜单：TR069管理
		tr069Menu := model.SysBaseMenu{
			ParentId:  0,
			Path:      "tr069",
			Name:      "tr069",
			Hidden:    false,
			Component: "view/routerHolder.vue",
			Sort:      10,
			Meta:      model.Meta{Title: "TR069管理", Icon: "monitor"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", tr069Menu.Name).FirstOrCreate(&tr069Menu).Error; err != nil {
			return err
		}

		// 二级菜单：设备列表
		deviceMenu := model.SysBaseMenu{
			ParentId:  tr069Menu.ID,
			Path:      "device",
			Name:      "tr069Device",
			Hidden:    false,
			Component: "plugin/tr069/view/device/index.vue",
			Sort:      1,
			Meta:      model.Meta{Title: "设备列表", Icon: "list"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", deviceMenu.Name).FirstOrCreate(&deviceMenu).Error; err != nil {
			return err
		}

		commandRecordMenu := model.SysBaseMenu{
			ParentId:  tr069Menu.ID,
			Path:      "commandRecord",
			Name:      "tr069CommandRecord",
			Hidden:    false,
			Component: "plugin/tr069/view/command-record/index.vue",
			Sort:      2,
			Meta:      model.Meta{Title: "RPC 记录", Icon: "document"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", commandRecordMenu.Name).FirstOrCreate(&commandRecordMenu).Error; err != nil {
			return err
		}
		commandRecordMenu.ParentId = tr069Menu.ID
		commandRecordMenu.Component = "plugin/tr069/view/command-record/index.vue"
		commandRecordMenu.Path = "commandRecord"
		commandRecordMenu.Sort = 2
		commandRecordMenu.Meta.Title = "RPC 记录"
		commandRecordMenu.Meta.Icon = "document"
		commandRecordMenu.Hidden = false
		if err := tx.Save(&commandRecordMenu).Error; err != nil {
			return err
		}

		// 二级菜单：告警管理（作为父菜单）
		alarmMenu := model.SysBaseMenu{
			ParentId:  tr069Menu.ID,
			Path:      "alarm",
			Name:      "tr069Alarm",
			Hidden:    false,
			Component: "view/routerHolder.vue",
			Sort:      3,
			Meta:      model.Meta{Title: "告警管理", Icon: "bell"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", alarmMenu.Name).FirstOrCreate(&alarmMenu).Error; err != nil {
			return err
		}
		// 强制更新字段，确保作为父菜单配置正确
		alarmMenu.ParentId = tr069Menu.ID
		alarmMenu.Component = "view/routerHolder.vue"
		alarmMenu.Path = "alarm"
		alarmMenu.Sort = 3
		alarmMenu.Meta.Title = "告警管理"
		alarmMenu.Meta.Icon = "bell"
		alarmMenu.Hidden = false
		if err := tx.Save(&alarmMenu).Error; err != nil {
			return err
		}

		// 三级菜单：当前告警
		activeAlarmMenu := model.SysBaseMenu{
			ParentId:  alarmMenu.ID,
			Path:      "activeAlarm",
			Name:      "tr069ActiveAlarm",
			Hidden:    false,
			Component: "plugin/tr069/view/alarm/active.vue",
			Sort:      1,
			Meta:      model.Meta{Title: "当前告警", Icon: "warning"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", activeAlarmMenu.Name).FirstOrCreate(&activeAlarmMenu).Error; err != nil {
			return err
		}
		// 强制更新字段
		activeAlarmMenu.ParentId = alarmMenu.ID
		activeAlarmMenu.Component = "plugin/tr069/view/alarm/active.vue"
		activeAlarmMenu.Path = "activeAlarm"
		activeAlarmMenu.Sort = 1
		activeAlarmMenu.Meta.Title = "当前告警"
		activeAlarmMenu.Meta.Icon = "warning"
		activeAlarmMenu.Hidden = false
		if err := tx.Save(&activeAlarmMenu).Error; err != nil {
			return err
		}

		// 三级菜单：历史告警
		historyAlarmMenu := model.SysBaseMenu{
			ParentId:  alarmMenu.ID,
			Path:      "historyAlarm",
			Name:      "tr069HistoryAlarm",
			Hidden:    false,
			Component: "plugin/tr069/view/alarm/history.vue",
			Sort:      2,
			Meta:      model.Meta{Title: "历史告警", Icon: "document"},
		}
		if err := tx.Model(&model.SysBaseMenu{}).Where("name = ?", historyAlarmMenu.Name).FirstOrCreate(&historyAlarmMenu).Error; err != nil {
			return err
		}
		// 强制更新字段
		historyAlarmMenu.ParentId = alarmMenu.ID
		historyAlarmMenu.Component = "plugin/tr069/view/alarm/history.vue"
		historyAlarmMenu.Path = "historyAlarm"
		historyAlarmMenu.Sort = 2
		historyAlarmMenu.Meta.Title = "历史告警"
		historyAlarmMenu.Meta.Icon = "document"
		historyAlarmMenu.Hidden = false
		if err := tx.Save(&historyAlarmMenu).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		global.GVA_LOG.Error("注册菜单失败", zap.Error(err))
	}
}
