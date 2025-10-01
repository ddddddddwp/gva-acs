package initialize

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// InitializeTR069Server 初始化TR069服务器
func InitializeTR069Server() {
	if !tr069global.TR069Config.Enabled {
		global.GVA_LOG.Info("TR069服务器未启用")
		return
	}

	// 创建TR069服务器
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 注册TR069请求处理路由
	router.POST("/tr069", func(c *gin.Context) {
		// 读取请求体
		body, err := c.GetRawData()
		if err != nil {
			global.GVA_LOG.Error("读取TR069请求体失败", zap.Error(err))
			c.Status(400)
			return
		}
		
		// 使用tr069-core处理请求
		resp, err := tr069global.TR069Parser.ParseInform(body)
		if err != nil {
			global.GVA_LOG.Error("解析TR069请求失败", zap.Error(err))
			c.Status(500)
			return
		}
		
		// 构建响应
		response := tr069global.TR069Builder.BuildInformResponse(resp)
		
		// 触发事件
		tr069global.TR069EventManager.TriggerEvent("inform_received", map[string]interface{}{
			"deviceId": resp.DeviceID,
			"timestamp": time.Now(),
		})
		
		// 设置响应头
		c.Header("Content-Type", "text/xml; charset=utf-8")
		c.Writer.Write(response)
	})
	
	// 启动TR069服务器
	addr := fmt.Sprintf(":%d", tr069global.TR069Config.ServerPort)
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