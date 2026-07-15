package infolog

import (
	"os"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"go.uber.org/zap"
)

func Write(s string) {
	settings := config.CurrentRuntime().Settings
	if !settings.InfoLogEnable {
		return
	}
	dir := settings.InfoLogDir
	if dir == "" {
		dir = "./log"
	}
	date := time.Now().Format("2006-01-02")
	base := dir + "/" + date
	if err := os.MkdirAll(base, 0o755); err != nil {
		if global.GVA_LOG != nil {
			global.GVA_LOG.Error("TR069 infolog mkdir failed", zap.Error(err), zap.String("dir", base))
		}
		return
	}
	path := base + "/tr069info.log"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		if global.GVA_LOG != nil {
			global.GVA_LOG.Error("TR069 infolog open failed", zap.Error(err), zap.String("path", path))
		}
		return
	}
	defer f.Close()
	_, _ = f.WriteString(s)
	_, _ = f.WriteString("\n")
}
