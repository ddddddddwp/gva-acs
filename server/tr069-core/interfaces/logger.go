// Package interfaces defines the interfaces for TR069 protocol library.
package interfaces

// LogLevel 定义日志级别
type LogLevel int

const (
// LogLevelDebug 调试级别
LogLevelDebug LogLevel = iota
// LogLevelInfo 信息级别
LogLevelInfo
// LogLevelWarn 警告级别
LogLevelWarn
// LogLevelError 错误级别
LogLevelError
// LogLevelFatal 致命错误级别
LogLevelFatal
)

// LogFormat 定义日志格式类型
type LogFormat string

const (
// LogFormatText 文本格式
LogFormatText LogFormat = "text"
// LogFormatJSON JSON格式
LogFormatJSON LogFormat = "json"
// LogFormatCustom 自定义格式
LogFormatCustom LogFormat = "custom"
// LogFormatTabSeparated Tab分隔格式
LogFormatTabSeparated LogFormat = "tab"
// LogFormatCompact 紧凑格式（时间 级别 调用者 消息）
LogFormatCompact LogFormat = "compact"
)

// LogField 定义日志字段
type LogField struct {
Key   string
Value interface{}
}

// Logger 定义日志记录接口
type Logger interface {
// Debug 记录调试级别日志
Debug(msg string, fields ...LogField)
// Info 记录信息级别日志
Info(msg string, fields ...LogField)
// Warn 记录警告级别日志
Warn(msg string, fields ...LogField)
// Error 记录错误级别日志
Error(msg string, fields ...LogField)
// Fatal 记录致命错误级别日志
Fatal(msg string, fields ...LogField)

// WithFields 创建带有固定字段的新日志记录器
WithFields(fields ...LogField) Logger
// WithContext 创建带有上下文的新日志记录器
WithContext(ctx interface{}) Logger

// SetLevel 设置日志级别
SetLevel(level LogLevel)
// GetLevel 获取当前日志级别
GetLevel() LogLevel
// SetFormat 设置日志格式
SetFormat(format LogFormat)
// GetFormat 获取当前日志格式
GetFormat() LogFormat
// SetOutput 设置日志输出目标
SetOutput(output interface{})
}

// LoggerOption 定义日志记录器配置选项
type LoggerOption func(Logger)
