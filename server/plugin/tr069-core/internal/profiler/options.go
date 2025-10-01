// Package profiler provides implementation for performance profiling tools.
package profiler

import (
"github.com/root/demo/tr069/interfaces"
)

// WithOutputDirectory 设置性能分析输出目录
func WithOutputDirectory(dir string) interfaces.ProfilerOption {
return func(p interfaces.Profiler) {
if profiler, ok := p.(*defaultProfiler); ok {
profiler.outputDir = dir
}
}
}

// WithMemoryProfiling 设置是否启用内存分析
func WithMemoryProfiling(enable bool) interfaces.ProfilerOption {
return func(p interfaces.Profiler) {
if profiler, ok := p.(*defaultProfiler); ok {
profiler.enableMemoryProfiling = enable
}
}
}

// WithCPUProfiling 设置是否启用CPU分析
func WithCPUProfiling(enable bool) interfaces.ProfilerOption {
return func(p interfaces.Profiler) {
if profiler, ok := p.(*defaultProfiler); ok {
profiler.enableCPUProfiling = enable
}
}
}

// WithBenchmarkIterations 设置基准测试迭代次数
func WithBenchmarkIterations(iterations int) interfaces.ProfilerOption {
return func(p interfaces.Profiler) {
if profiler, ok := p.(*defaultProfiler); ok {
profiler.benchmarkIterations = iterations
}
}
}
