// Package profiler provides implementation for performance profiling tools.
package profiler

import "github.com/root/demo/tr069/interfaces"

// NewProfilerFactory 创建性能分析工具工厂函数
func NewProfilerFactory() func(options ...interfaces.ProfilerOption) interfaces.Profiler {
return func(options ...interfaces.ProfilerOption) interfaces.Profiler {
return NewDefaultProfiler(options...)
}
}
