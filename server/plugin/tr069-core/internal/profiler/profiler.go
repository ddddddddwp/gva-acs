// Package profiler provides implementation for performance profiling tools.
package profiler

import (
"bytes"
"encoding/json"
"fmt"
"os"
"path/filepath"
"runtime"
"runtime/pprof"
"time"

"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// defaultProfiler 实现性能分析工具接口
type defaultProfiler struct {
// 活跃的性能分析
activeProfiles map[interfaces.ProfileType]*activeProfile
// 性能分析结果
results map[interfaces.ProfileType]*interfaces.ProfileResult
// 性能分析输出目录
outputDir string
// 是否启用内存分析
enableMemoryProfiling bool
// 是否启用CPU分析
enableCPUProfiling bool
// 基准测试迭代次数
benchmarkIterations int
}

// activeProfile 表示活跃的性能分析
type activeProfile struct {
// 性能分析类型
profileType interfaces.ProfileType
// 开始时间
startTime time.Time
// 缓冲区
buffer *bytes.Buffer
}

// NewDefaultProfiler 创建一个新的默认性能分析工具
func NewDefaultProfiler(options ...interfaces.ProfilerOption) interfaces.Profiler {
p := &defaultProfiler{
activeProfiles:        make(map[interfaces.ProfileType]*activeProfile),
results:               make(map[interfaces.ProfileType]*interfaces.ProfileResult),
outputDir:             os.TempDir(),
enableMemoryProfiling: true,
enableCPUProfiling:    true,
benchmarkIterations:   1000,
}

for _, option := range options {
option(p)
}

return p
}

// StartProfile 开始性能分析
func (p *defaultProfiler) StartProfile(profileType interfaces.ProfileType) error {
// 检查是否已经有活跃的性能分析
if _, exists := p.activeProfiles[profileType]; exists {
return fmt.Errorf("profile of type %s is already active", profileType)
}

buffer := &bytes.Buffer{}
active := &activeProfile{
profileType: profileType,
startTime:   time.Now(),
buffer:      buffer,
}

var err error
switch profileType {
case interfaces.ProfileTypeCPU:
if !p.enableCPUProfiling {
return fmt.Errorf("CPU profiling is disabled")
}
err = pprof.StartCPUProfile(buffer)
case interfaces.ProfileTypeMemory:
if !p.enableMemoryProfiling {
return fmt.Errorf("memory profiling is disabled")
}
// 内存分析在停止时收集
case interfaces.ProfileTypeBlock:
runtime.SetBlockProfileRate(1)
case interfaces.ProfileTypeGoroutine:
// 协程分析在停止时收集
default:
return fmt.Errorf("unsupported profile type: %s", profileType)
}

if err != nil {
return err
}

p.activeProfiles[profileType] = active
return nil
}

// StopProfile 停止性能分析并返回结果
func (p *defaultProfiler) StopProfile(profileType interfaces.ProfileType) (*interfaces.ProfileResult, error) {
active, exists := p.activeProfiles[profileType]
if !exists {
return nil, fmt.Errorf("no active profile of type %s", profileType)
}

endTime := time.Now()

switch profileType {
case interfaces.ProfileTypeCPU:
pprof.StopCPUProfile()
case interfaces.ProfileTypeMemory:
err := pprof.WriteHeapProfile(active.buffer)
if err != nil {
return nil, err
}
case interfaces.ProfileTypeBlock:
runtime.SetBlockProfileRate(0)
err := pprof.Lookup("block").WriteTo(active.buffer, 0)
if err != nil {
return nil, err
}
case interfaces.ProfileTypeGoroutine:
err := pprof.Lookup("goroutine").WriteTo(active.buffer, 0)
if err != nil {
return nil, err
}
}

// 创建结果
result := &interfaces.ProfileResult{
ProfileType: profileType,
StartTime:   active.startTime,
EndTime:     endTime,
Data:        active.buffer.Bytes(),
Metrics:     p.collectMetrics(profileType),
}

// 保存结果
p.results[profileType] = result

// 删除活跃的性能分析
delete(p.activeProfiles, profileType)

return result, nil
}

// GetProfile 获取性能分析结果
func (p *defaultProfiler) GetProfile(profileType interfaces.ProfileType) (*interfaces.ProfileResult, error) {
result, exists := p.results[profileType]
if !exists {
return nil, fmt.Errorf("no profile result of type %s", profileType)
}
return result, nil
}

// ExportProfile 导出性能分析结果
func (p *defaultProfiler) ExportProfile(profileType interfaces.ProfileType, format string) ([]byte, error) {
result, err := p.GetProfile(profileType)
if err != nil {
return nil, err
}

switch format {
case "json":
return json.Marshal(result.Metrics)
case "raw":
return result.Data, nil
case "file":
filename := filepath.Join(p.outputDir, fmt.Sprintf("%s-%d.prof", profileType, time.Now().Unix()))
err := os.WriteFile(filename, result.Data, 0644)
if err != nil {
return nil, err
}
return []byte(filename), nil
default:
return nil, fmt.Errorf("unsupported export format: %s", format)
}
}

// MemoryStats 获取内存使用统计
func (p *defaultProfiler) MemoryStats() map[string]interface{} {
var memStats runtime.MemStats
runtime.ReadMemStats(&memStats)

return map[string]interface{}{
"alloc":              memStats.Alloc,
"total_alloc":        memStats.TotalAlloc,
"sys":                memStats.Sys,
"lookups":            memStats.Lookups,
"mallocs":            memStats.Mallocs,
"frees":              memStats.Frees,
"heap_alloc":         memStats.HeapAlloc,
"heap_sys":           memStats.HeapSys,
"heap_idle":          memStats.HeapIdle,
"heap_inuse":         memStats.HeapInuse,
"heap_released":      memStats.HeapReleased,
"heap_objects":       memStats.HeapObjects,
"stack_inuse":        memStats.StackInuse,
"stack_sys":          memStats.StackSys,
"next_gc":            memStats.NextGC,
"last_gc":            memStats.LastGC,
"gc_cpu_fraction":    memStats.GCCPUFraction,
"num_gc":             memStats.NumGC,
"num_forced_gc":      memStats.NumForcedGC,
"gc_pause_total_ns":  memStats.PauseTotalNs,
"gc_pause_ns":        memStats.PauseNs[(memStats.NumGC+255)%256],
"gc_pause_end":       memStats.PauseEnd[(memStats.NumGC+255)%256],
}
}

// CPUStats 获取CPU使用统计
func (p *defaultProfiler) CPUStats() map[string]interface{} {
return map[string]interface{}{
"num_cpu":        runtime.NumCPU(),
"num_goroutine":  runtime.NumGoroutine(),
"num_cgo_call":   runtime.NumCgoCall(),
"gomaxprocs":     runtime.GOMAXPROCS(0),
"compiler":       runtime.Compiler,
"os":             runtime.GOOS,
"arch":           runtime.GOARCH,
"max_threads":    runtime.GOMAXPROCS(0),
}
}

// RunBenchmark 运行基准测试
func (p *defaultProfiler) RunBenchmark(name string, fn func()) *interfaces.BenchmarkResult {
result := &interfaces.BenchmarkResult{
Name:       name,
Iterations: p.benchmarkIterations,
}

// 运行垃圾回收以获得更准确的内存分配测量
runtime.GC()

var memStatsBefore, memStatsAfter runtime.MemStats
runtime.ReadMemStats(&memStatsBefore)

startTime := time.Now()

// 运行基准测试
for i := 0; i < p.benchmarkIterations; i++ {
fn()
}

result.TotalTime = time.Since(startTime)
result.AverageTime = result.TotalTime / time.Duration(p.benchmarkIterations)

runtime.ReadMemStats(&memStatsAfter)

// 计算内存分配
result.AllocatedBytes = int64(memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc)
result.AllocationsCount = int64(memStatsAfter.Mallocs - memStatsBefore.Mallocs)

return result
}

// collectMetrics 收集性能指标
func (p *defaultProfiler) collectMetrics(profileType interfaces.ProfileType) map[string]interface{} {
metrics := make(map[string]interface{})

switch profileType {
case interfaces.ProfileTypeCPU:
for k, v := range p.CPUStats() {
metrics[k] = v
}
case interfaces.ProfileTypeMemory:
for k, v := range p.MemoryStats() {
metrics[k] = v
}
case interfaces.ProfileTypeBlock:
metrics["num_goroutine"] = runtime.NumGoroutine()
case interfaces.ProfileTypeGoroutine:
metrics["num_goroutine"] = runtime.NumGoroutine()
}

return metrics
}
