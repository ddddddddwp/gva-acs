package initialize

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/handler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func StartTR069Server() {
	addr := tr069Global.GlobalConfig.Address
	if addr == "" {
		addr = ":7547" // Default port
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	// Register CWMP Handler
	engine.POST("/", handler.CWMPHandler)
	engine.POST("/acs", handler.CWMPHandler)

	go func() {
		global.GVA_LOG.Info("Starting TR069 Server", zap.String("address", addr))
		if err := engine.Run(addr); err != nil {
			global.GVA_LOG.Error("TR069 Server failed to start", zap.Error(err))
		}
	}()
}
