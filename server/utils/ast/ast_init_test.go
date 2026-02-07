package ast

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"path/filepath"
)

func init() {
	global.GVA_CONFIG.AutoCode.Root, _ = filepath.Abs("../../../")
	global.GVA_CONFIG.AutoCode.Server = "server"
}
