// Package logger 提供TR069库的日志组件实现
package logger

import (
	"io"
	"os"

	"github.com/root/demo/tr069/interfaces"
)

// Option 定义日志记录器的配置选项
type Option func(*options)

// options 包含日志记录器的配置
type options struct {
	level  interfaces.LogLevel
	format interfaces.LogFormat
	output io.Writer
}

// WithLogLevel 设置日志级别
func WithLogLevel(level interfaces.LogLevel) Option {
	return func(opts *options) {
		opts.level = level
	}
}

// WithLogFormat 设置日志格式
func WithLogFormat(format interfaces.LogFormat) Option {
	return func(opts *options) {
		opts.format = format
	}
}

// WithLogOutput 设置日志输出目标
func WithLogOutput(output io.Writer) Option {
	return func(opts *options) {
		opts.output = output
	}
}

// defaultOptions 返回默认配置
func defaultOptions() *options {
	return &options{
		level:  interfaces.LogLevelInfo,
		format: interfaces.LogFormatJSON,
		output: os.Stdout,
	}
}

// applyOptions 应用配置选项
func applyOptions(opts *options, options ...Option) {
	for _, opt := range options {
		opt(opts)
	}
}