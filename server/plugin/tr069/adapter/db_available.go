package adapter

import "github.com/ddddddddwp/gva-acs/server/global"

func DBAvailable() bool { return global.GVA_DB != nil }
