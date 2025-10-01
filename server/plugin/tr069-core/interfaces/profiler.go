// Package interfaces defines the interfaces for TR069 protocol library.
package interfaces

import "time"

// ProfileType 定义性能分析类型
type ProfileType string

const (
// ProfileTypeCPU CPU性能分析
ProfileTypeCPU ProfileType = "cpu"
// ProfileTypeMemory 内存性能分析
ProfileTypeMemory ProfileType = "memory"
// ProfileTypeBlock 阻塞性能分析
ProfileTypeBlock ProfileType = "block"
// ProfileTypeGoroutine 协程性能分析
ProfileTypeGoroutine ProfileType = "goroutine"
)

// ProfileResult 定义性能分析结果
type ProfileResult struct {
// ProfileType 性能分析类型
ProfileType ProfileType
// StartTime 开始时间
StartTime time.Time
// EndTime 结束时间
EndTime time.Time
// Data 性能分析数据
Data []byte
// Metrics 性能指标
Metrics map[string]interface{}
}

// Profiler 定义性能分析工具接口
type Profiler interface {
// StartProfile 开始性能分析
StartProfile(profileType ProfileType) error

// StopProfile 停止性能分析并返回结果
StopProfile(profileType ProfileType) (*ProfileResult, error)

// GetProfile 获取性能分析结果
GetProfile(profileType ProfileType) (*ProfileResult, error)

// ExportProfile 导出性能分析结果
ExportProfile(profileType ProfileType, format string) ([]byte, error)

// MemoryStats 获取内存使用统计
MemoryStats() map[string]interface{}

// CPUStats 获取CPU使用统计
CPUStats() map[string]interface{}

// RunBenchmark 运行基准测试
RunBenchmark(name string, fn func()) *BenchmarkResult
}

// BenchmarkResult 定义基准测试结果
type BenchmarkResult struct {
// Name 基准测试名称
Name string
// Iterations 迭代次数
Iterations int
// TotalTime 总时间
TotalTime time.Duration
// AverageTime 平均时间
AverageTime time.Duration
// AllocatedBytes 分配的字节数
AllocatedBytes int64
// AllocationsCount 分配次数
AllocationsCount int64
}

// ProfilerOption 定义性能分析工具配置选项
type ProfilerOption func(Profiler)
