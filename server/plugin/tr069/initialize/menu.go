package initialize

import (
	"context"
	model "github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/ddddddddwp/gva-acs/server/plugin/plugin-tool/utils"
)

func Menu(ctx context.Context) {
	entities := []model.SysBaseMenu{
		{
			ParentId:  0,
			Path:      "tr069",
			Name:      "tr069",
			Hidden:    false,
			Component: "view/routerHolder.vue",
			Sort:      10,
			Meta:      model.Meta{Title: "TR069管理", Icon: "monitor"},
		},
		{
			Path:      "device",
			Name:      "tr069Device",
			Hidden:    false,
			Component: "plugin/tr069/view/device/index.vue",
			Sort:      1,
			Meta:      model.Meta{Title: "设备列表", Icon: "list"},
		},
		{
			Path:      "alarm",
			Name:      "tr069Alarm",
			Hidden:    false,
			Component: "plugin/tr069/view/alarm/index.vue",
			Sort:      2,
			Meta:      model.Meta{Title: "告警管理", Icon: "bell"},
		},
	}
	utils.RegisterMenus(entities...)
}
