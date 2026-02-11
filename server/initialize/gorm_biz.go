package initialize

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Model "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
)

func bizModel() error {
	db := global.GVA_DB
	return db.AutoMigrate(
		tr069Model.Device{},
		tr069Model.Command{},
		tr069Model.DataModelValue{},
		tr069Model.FAPService{},
		tr069Model.DeviceRPCMethods{},
	)
}
