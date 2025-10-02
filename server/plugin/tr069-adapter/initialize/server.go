package initialize

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	adapterGlobal "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/service"
	"go.uber.org/zap"
)

// InitializeTR069Server 初始化TR069服务器
func InitializeTR069Server() {
	if !adapterGlobal.TR069Config.Enabled {
		global.GVA_LOG.Info("TR069服务器未启用")
		return
	}

	// 创建TR069服务器
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 注册TR069请求处理路由
	router.POST("/tr069", func(c *gin.Context) {
		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			global.GVA_LOG.Error("读取TR069请求体失败", zap.Error(err))
			c.Status(400)
			return
		}
		
		// 使用适配器服务处理请求
		ctx := context.Background()
		resp, err := service.ServiceGroupApp.ProcessInform(ctx, body)
		if err != nil {
			global.GVA_LOG.Error("处理TR069请求失败", zap.Error(err))
			c.Status(500)
			return
		}
		
		// 设置响应头
		c.Header("Content-Type", "text/xml; charset=utf-8")
		c.Writer.Write(resp)
	})
	
	// 启动TR069服务器
	addr := fmt.Sprintf(":%d", adapterGlobal.TR069Config.ServerPort)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	
	go func() {
		global.GVA_LOG.Info("TR069服务器启动", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			global.GVA_LOG.Error("TR069服务器启动失败", zap.Error(err))
		}
	}()
}