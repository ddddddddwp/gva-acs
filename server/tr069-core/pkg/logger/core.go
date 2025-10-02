// Package logger 提供TR069库的日志组件实现
package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// InitCoreLogger 初始化TR069 Core日志系统
// 确保日志目录存在并设置适当的权限
func InitCoreLogger() error {
	config := DefaultCoreLogStorageConfig()

	// 确保日志目录存在
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 检查日志文件权限
	if _, err := os.Stat(config.FilePath); err == nil {
		// 文件存在，检查权限
		if err := os.Chmod(config.FilePath, 0644); err != nil {
			return fmt.Errorf("设置日志文件权限失败: %v", err)
		}
	}

	// 创建并设置全局日志记录器
	logger := NewFileLogger(config)
	SetGlobalLogger(logger)

	// 记录初始化信息
	Info("TR069 Core日志系统初始化完成",
		interfaces.LogField{Key: "log_file", Value: config.FilePath},
		interfaces.LogField{Key: "max_size", Value: config.MaxSize},
		interfaces.LogField{Key: "max_backups", Value: config.MaxBackups},
		interfaces.LogField{Key: "max_age", Value: config.MaxAge},
	)

	return nil
}

// GetCoreLogger 获取TR069 Core专用的日志记录器
func GetCoreLogger() interfaces.Logger {
	return DefaultCoreLogger()
}

// LogCoreInfo 记录Core信息日志
func LogCoreInfo(msg string, fields ...interfaces.LogField) {
	coreFields := append([]interfaces.LogField{
		{Key: "component", Value: "tr069-core"},
	}, fields...)
	Info(msg, coreFields...)
}

// LogCoreError 记录Core错误日志
func LogCoreError(msg string, fields ...interfaces.LogField) {
	coreFields := append([]interfaces.LogField{
		{Key: "component", Value: "tr069-core"},
	}, fields...)
	Error(msg, coreFields...)
}

// LogCoreDebug 记录Core调试日志
func LogCoreDebug(msg string, fields ...interfaces.LogField) {
	coreFields := append([]interfaces.LogField{
		{Key: "component", Value: "tr069-core"},
	}, fields...)
	Debug(msg, coreFields...)
}

// LogCoreWarn 记录Core警告日志
func LogCoreWarn(msg string, fields ...interfaces.LogField) {
	coreFields := append([]interfaces.LogField{
		{Key: "component", Value: "tr069-core"},
	}, fields...)
	Warn(msg, coreFields...)
}
