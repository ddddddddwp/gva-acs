package initialize

import (
	"github.com/ddddddddwp/gva-acs/server/global"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate()
	if err != nil {
		return err
	}
	return nil
}
