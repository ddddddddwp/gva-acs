package initialize

import (
	"context"
	model "github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/ddddddddwp/gva-acs/server/plugin/plugin-tool/utils"
)

func Dictionary(ctx context.Context) {
	entities := []model.SysDictionary{}
	utils.RegisterDictionaries(entities...)
}
