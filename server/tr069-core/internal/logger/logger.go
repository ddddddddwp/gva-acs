// Package logger provides implementation for detailed logging functionality.
package logger

import (
"encoding/json"
"fmt"
"io"
"os"
"sync"
"time"

"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// defaultLogger 实现了interfaces.Logger接口
type defaultLogger struct {
mu     sync.Mutex
level  interfaces.LogLevel
format interfaces.LogFormat
output io.Writer
fields []interfaces.LogField
ctx    interface{}
}

// NewDefaultLogger 创建一个新的默认日志记录器
func NewDefaultLogger(options ...interfaces.LoggerOption) interfaces.Logger {
logger := &defaultLogger{
level:  interfaces.LogLevelInfo,
format: interfaces.LogFormatText,
output: os.Stdout,
fields: make([]interfaces.LogField, 0),
}

for _, option := range options {
option(logger)
}

return logger
}

// Debug 实现Debug级别日志记录
func (l *defaultLogger) Debug(msg string, fields ...interfaces.LogField) {
if l.level <= interfaces.LogLevelDebug {
l.log(interfaces.LogLevelDebug, msg, fields...)
}
}

// Info 实现Info级别日志记录
func (l *defaultLogger) Info(msg string, fields ...interfaces.LogField) {
if l.level <= interfaces.LogLevelInfo {
l.log(interfaces.LogLevelInfo, msg, fields...)
}
}

// Warn 实现Warn级别日志记录
func (l *defaultLogger) Warn(msg string, fields ...interfaces.LogField) {
if l.level <= interfaces.LogLevelWarn {
l.log(interfaces.LogLevelWarn, msg, fields...)
}
}

// Error 实现Error级别日志记录
func (l *defaultLogger) Error(msg string, fields ...interfaces.LogField) {
if l.level <= interfaces.LogLevelError {
l.log(interfaces.LogLevelError, msg, fields...)
}
}

// Fatal 实现Fatal级别日志记录
func (l *defaultLogger) Fatal(msg string, fields ...interfaces.LogField) {
if l.level <= interfaces.LogLevelFatal {
l.log(interfaces.LogLevelFatal, msg, fields...)
os.Exit(1)
}
}

// WithFields 创建带有固定字段的新日志记录器
func (l *defaultLogger) WithFields(fields ...interfaces.LogField) interfaces.Logger {
newLogger := &defaultLogger{
level:  l.level,
format: l.format,
output: l.output,
ctx:    l.ctx,
}

// 复制现有字段
newLogger.fields = make([]interfaces.LogField, len(l.fields)+len(fields))
copy(newLogger.fields, l.fields)
copy(newLogger.fields[len(l.fields):], fields)

return newLogger
}

// WithContext 创建带有上下文的新日志记录器
func (l *defaultLogger) WithContext(ctx interface{}) interfaces.Logger {
newLogger := &defaultLogger{
level:  l.level,
format: l.format,
output: l.output,
ctx:    ctx,
}

// 复制现有字段
newLogger.fields = make([]interfaces.LogField, len(l.fields))
copy(newLogger.fields, l.fields)

return newLogger
}

// SetLevel 设置日志级别
func (l *defaultLogger) SetLevel(level interfaces.LogLevel) {
l.mu.Lock()
defer l.mu.Unlock()
l.level = level
}

// GetLevel 获取当前日志级别
func (l *defaultLogger) GetLevel() interfaces.LogLevel {
l.mu.Lock()
defer l.mu.Unlock()
return l.level
}

// SetFormat 设置日志格式
func (l *defaultLogger) SetFormat(format interfaces.LogFormat) {
l.mu.Lock()
defer l.mu.Unlock()
l.format = format
}

// GetFormat 获取当前日志格式
func (l *defaultLogger) GetFormat() interfaces.LogFormat {
l.mu.Lock()
defer l.mu.Unlock()
return l.format
}

// SetOutput 设置日志输出目标
func (l *defaultLogger) SetOutput(output interface{}) {
l.mu.Lock()
defer l.mu.Unlock()
if writer, ok := output.(io.Writer); ok {
l.output = writer
}
}

// log 内部日志记录方法
func (l *defaultLogger) log(level interfaces.LogLevel, msg string, fields ...interfaces.LogField) {
l.mu.Lock()
defer l.mu.Unlock()

timestamp := time.Now().Format(time.RFC3339)
levelStr := l.levelToString(level)

// 合并所有字段
allFields := make([]interfaces.LogField, 0, len(l.fields)+len(fields))
allFields = append(allFields, l.fields...)
allFields = append(allFields, fields...)

switch l.format {
case interfaces.LogFormatJSON:
l.logJSON(timestamp, levelStr, msg, allFields)
case interfaces.LogFormatText:
fallthrough
default:
l.logText(timestamp, levelStr, msg, allFields)
}
}

// levelToString 将日志级别转换为字符串
func (l *defaultLogger) levelToString(level interfaces.LogLevel) string {
switch level {
case interfaces.LogLevelDebug:
return "DEBUG"
case interfaces.LogLevelInfo:
return "INFO"
case interfaces.LogLevelWarn:
return "WARN"
case interfaces.LogLevelError:
return "ERROR"
case interfaces.LogLevelFatal:
return "FATAL"
default:
return "UNKNOWN"
}
}

// logText 以文本格式记录日志
func (l *defaultLogger) logText(timestamp, level, msg string, fields []interfaces.LogField) {
logLine := fmt.Sprintf("%s [%s] %s", timestamp, level, msg)

if len(fields) > 0 {
logLine += " {"
for i, field := range fields {
if i > 0 {
logLine += ", "
}
logLine += fmt.Sprintf("%s=%v", field.Key, field.Value)
}
logLine += "}"
}

fmt.Fprintln(l.output, logLine)
}

// logJSON 以JSON格式记录日志
func (l *defaultLogger) logJSON(timestamp, level, msg string, fields []interfaces.LogField) {
logData := map[string]interface{}{
"timestamp": timestamp,
"level":     level,
"message":   msg,
}

if l.ctx != nil {
logData["context"] = l.ctx
}

if len(fields) > 0 {
fieldsMap := make(map[string]interface{})
for _, field := range fields {
fieldsMap[field.Key] = field.Value
}
logData["fields"] = fieldsMap
}

jsonData, err := json.Marshal(logData)
if err != nil {
fmt.Fprintf(l.output, "{\"error\":\"failed to marshal log entry: %v\"}\n", err)
return
}

fmt.Fprintln(l.output, string(jsonData))
}
