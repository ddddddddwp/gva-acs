// Package monitor provides implementation for monitoring functionality.
package monitor

import (
"encoding/json"
"fmt"
"os"
"path/filepath"

"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// Exporter 定义了指标导出器接口
type Exporter interface {
// Export 导出指标
Export(snapshot *interfaces.MetricSnapshot) error
}

// FileExporter 实现了基于文件的指标导出器
type FileExporter struct {
directory string
format    string
prefix    string
}

// NewFileExporter 创建一个新的文件导出器
func NewFileExporter(directory, format, prefix string) (Exporter, error) {
// 确保目录存在
if err := os.MkdirAll(directory, 0755); err != nil {
return nil, fmt.Errorf("failed to create directory: %w", err)
}

return &FileExporter{
directory: directory,
format:    format,
prefix:    prefix,
}, nil
}

// Export 将指标导出到文件
func (e *FileExporter) Export(snapshot *interfaces.MetricSnapshot) error {
if snapshot == nil {
return fmt.Errorf("snapshot cannot be nil")
}

var data []byte
var err error

// 根据格式导出
switch e.format {
case "json":
data, err = json.MarshalIndent(snapshot, "", "  ")
default:
data, err = json.MarshalIndent(snapshot, "", "  ") // 默认使用JSON格式
}

if err != nil {
return fmt.Errorf("failed to marshal snapshot: %w", err)
}

// 生成文件名
timestamp := snapshot.Timestamp.Format("20060102-150405")
filename := fmt.Sprintf("%s%s.%s", e.prefix, timestamp, e.format)
filepath := filepath.Join(e.directory, filename)

// 写入文件
if err := os.WriteFile(filepath, data, 0644); err != nil {
return fmt.Errorf("failed to write file: %w", err)
}

return nil
}

// MemoryExporter 实现了基于内存的指标导出器，主要用于测试
type MemoryExporter struct {
snapshots []*interfaces.MetricSnapshot
}

// NewMemoryExporter 创建一个新的内存导出器
func NewMemoryExporter() *MemoryExporter {
return &MemoryExporter{
snapshots: make([]*interfaces.MetricSnapshot, 0),
}
}

// Export 将指标保存到内存
func (e *MemoryExporter) Export(snapshot *interfaces.MetricSnapshot) error {
if snapshot == nil {
return fmt.Errorf("snapshot cannot be nil")
}

// 创建快照的副本
metricsCopy := make([]interfaces.MetricValue, len(snapshot.Metrics))
copy(metricsCopy, snapshot.Metrics)

snapshotCopy := &interfaces.MetricSnapshot{
Metrics:   metricsCopy,
Timestamp: snapshot.Timestamp,
}

e.snapshots = append(e.snapshots, snapshotCopy)
return nil
}

// GetSnapshots 获取所有保存的快照
func (e *MemoryExporter) GetSnapshots() []*interfaces.MetricSnapshot {
return e.snapshots
}

// Clear 清除所有快照
func (e *MemoryExporter) Clear() {
e.snapshots = make([]*interfaces.MetricSnapshot, 0)
}
