package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

// initServer 启动服务并实现优雅关闭
func initServer(address string, router *gin.Engine, readTimeout, writeTimeout time.Duration) {
	// 创建服务
	srv := &http.Server{
		Addr:           address,
		Handler:        router,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	// 在goroutine中启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("listen: %s\n", err)
			zap.L().Error("server启动失败", zap.Error(err))
			os.Exit(1)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	// kill (无参数) 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT
	// kill -9 发送 syscall.SIGKILL，但是无法被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("关闭WEB服务...")

	if err := shutdownServer(srv, utils.GlobalSystemEvents, 5*time.Second); err != nil {
		zap.L().Fatal("WEB服务关闭异常", zap.Error(err))
	}

	zap.L().Info("WEB服务已关闭")
}

func shutdownServer(srv server, events *utils.SystemEvents, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), timeout)
	var drainErr error
	if srv != nil {
		drainErr = srv.Shutdown(drainCtx)
	}
	cancelDrain()

	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), timeout)
	cleanupErr := events.TriggerShutdown(cleanupCtx)
	cancelCleanup()
	return errors.Join(drainErr, cleanupErr)
}
