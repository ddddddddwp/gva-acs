package initialize

import (
	"context"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMenuRegistersStableLogFileRouteBeforeAlarm(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(system.SysBaseMenu)); err != nil {
		t.Fatalf("migrate menus: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	Menu(context.Background())
	Menu(context.Background())

	var logMenu system.SysBaseMenu
	if err := db.Where("name = ?", "tr069LogFiles").First(&logMenu).Error; err != nil {
		t.Fatalf("load log file menu: %v", err)
	}
	if logMenu.Path != "logFiles" || logMenu.Component != "plugin/tr069/view/log-file/index.vue" || logMenu.Sort != 3 || logMenu.Hidden {
		t.Fatalf("log menu = %#v", logMenu)
	}
	var alarmMenu system.SysBaseMenu
	if err := db.Where("name = ?", "tr069Alarm").First(&alarmMenu).Error; err != nil {
		t.Fatalf("load alarm menu: %v", err)
	}
	if alarmMenu.Sort != 4 || alarmMenu.ParentId != logMenu.ParentId {
		t.Fatalf("alarm/log ordering = %d/%d", alarmMenu.Sort, logMenu.Sort)
	}
	var count int64
	if err := db.Model(new(system.SysBaseMenu)).Where("name = ?", "tr069LogFiles").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("log menu count = %d, err=%v", count, err)
	}
}
