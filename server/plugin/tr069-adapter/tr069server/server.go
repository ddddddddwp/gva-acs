package tr069server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/config"
	tr069request "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
)

// TR069Server TR069服务器
type TR069Server struct {
	config     *config.TR069Config
	db         *gorm.DB
	logger     *zap.Logger
	httpServer *http.Server
	soapHandler *SOAPHandler
}

// NewTR069Server 创建TR069服务器
func NewTR069Server(cfg *config.TR069Config, db *gorm.DB, logger *zap.Logger) *TR069Server {
	soapHandler := NewSOAPHandler(logger)
	
	return &TR069Server{
		config:        cfg,
		db:            db,
		logger:        logger,
		soapHandler:   soapHandler,
	}
}

// Start 启动TR069服务器
func (s *TR069Server) Start() error {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	
	// 创建路由
	router := gin.New()
	
	// 添加中间件
	router.Use(s.loggingMiddleware())
	router.Use(s.recoveryMiddleware())
	router.Use(s.corsMiddleware())
	
	// 注册路由
	s.registerRoutes(router)
	
	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(s.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(s.config.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}
	
	s.logger.Info("Starting TR069 server", zap.Int("port", s.config.Server.Port))
	
	// 启动服务器
	if s.config.Server.TLS.Enabled {
		return s.httpServer.ListenAndServeTLS(s.config.Server.TLS.CertFile, s.config.Server.TLS.KeyFile)
	}
	return s.httpServer.ListenAndServe()
}

// Stop 停止TR069服务器
func (s *TR069Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping TR069 server")
	return s.httpServer.Shutdown(ctx)
}

// registerRoutes 注册路由
func (s *TR069Server) registerRoutes(router *gin.Engine) {
	// TR069 CWMP endpoint
	router.POST("/", s.soapHandler.HandleCWMP)
	router.POST("/cwmp", s.soapHandler.HandleCWMP)
	
	// 健康检查
	router.GET("/health", s.healthCheck)
	
	// 设备状态查询（用于调试）
	api := router.Group("/api/v1")
	{
		api.GET("/devices", s.getDevices)
		api.GET("/devices/:id", s.getDevice)
		api.GET("/devices/:id/parameters", s.getDeviceParameters)
	}
}

// healthCheck 健康检查
func (s *TR069Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
		"service": "tr069-adapter",
	})
}

// getDevices 获取所有设备
func (s *TR069Server) getDevices(c *gin.Context) {
	// 使用默认的搜索参数
	req := tr069request.DeviceSearch{
		PageInfo: request.PageInfo{
			Page:     1,
			PageSize: 100,
		},
	}
	devices, _, err := s.deviceService.GetDeviceList(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// getDevice 获取单个设备
func (s *TR069Server) getDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}
	
	device, err := s.deviceService.GetDeviceByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device": device})
}

// getDeviceParameters 获取设备参数
func (s *TR069Server) getDeviceParameters(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}
	
	parameters, err := s.deviceService.GetParametersByDevice(uint(id), []string{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"parameters": parameters})
}

// loggingMiddleware 日志中间件
func (s *TR069Server) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		s.logger.Info("TR069 Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// recoveryMiddleware 恢复中间件
func (s *TR069Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		s.logger.Error("TR069 Server Panic", zap.Any("error", recovered))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	})
}

// corsMiddleware CORS中间件
func (s *TR069Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}