# TR069协议基础库核心接口使用指南

本文档提供TR069协议基础库的核心接口使用方法，帮助开发者快速上手使用本库进行TR069协议消息的解析和构建。

## 目录
1. [快速开始](#快速开始)
2. [核心接口](#核心接口)
   - [Parser接口](#parser接口)
   - [Builder接口](#builder接口)
3. [常用数据类型](#常用数据类型)
4. [使用示例](#使用示例)
5. [扩展功能](#扩展功能)

## 快速开始

### 安装
```bash
go get github.com/root/demo/tr069
```

### 基本使用
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/root/demo/tr069/factory"
)

func main() {
    // 创建解析器和构建器实例
    parser := factory.NewParser()
    builder := factory.NewBuilder()
    
    // 解析TR069消息
    msg, err := parser.ParseMessage(context.Background(), xmlData)
    if err != nil {
        panic(err)
    }
    
    // 构建响应消息
    response, err := builder.BuildMessage(context.Background(), msg)
    if err != nil {
        panic(err)
    }
    
    fmt.Println(string(response))
}
```

## 核心接口

### Parser接口

Parser接口负责解析TR069协议消息，从XML数据中提取结构化信息。

#### 主要方法

```go
// 解析完整TR069消息
ParseMessage(ctx context.Context, data []byte) (*Message, error)

// 仅解析RPC方法名
ParseRPCMethod(ctx context.Context, data []byte) (string, error)

// 解析参数列表
ParseParameterList(ctx context.Context, data []byte) ([]Parameter, error)

// 设置严格模式（更严格的验证）
SetStrictMode(strict bool)

// 获取当前严格模式状态
GetStrictMode() bool
```

#### 创建Parser实例

```go
// 使用工厂方法创建默认解析器
parser := factory.NewParser()

// 使用选项创建自定义解析器
parser := factory.NewParser(
    factory.WithStrictMode(true),
    factory.WithMaxDepth(100),
)
```

### Builder接口

Builder接口负责构建TR069协议消息，将结构化数据转换为XML格式。

#### 主要方法

```go
// 从Message结构体构建TR069消息
BuildMessage(ctx context.Context, msg *Message) ([]byte, error)

// 构建RPC响应
BuildRPCResponse(ctx context.Context, method string, params map[string]interface{}) ([]byte, error)

// 构建故障响应
BuildFault(ctx context.Context, code int, faultString string) ([]byte, error)

// 设置是否美化输出
SetPrettyPrint(pretty bool)

// 获取当前美化输出状态
GetPrettyPrint() bool
```

#### 创建Builder实例

```go
// 使用工厂方法创建默认构建器
builder := factory.NewBuilder()

// 使用选项创建自定义构建器
builder := factory.NewBuilder(
    factory.WithPrettyPrint(true),
    factory.WithNamespace("urn:dslforum-org:cwmp-1-0"),
)
```

## 常用数据类型

### Message

表示一个完整的TR069消息。

```go
type Message struct {
    ID         string      // 消息ID
    Method     string      // RPC方法名
    Parameters []Parameter // 参数列表
    Fault      *Fault      // 故障信息（如果有）
}
```

### Parameter

表示TR069消息中的参数。

```go
type Parameter struct {
    Name  string      // 参数名
    Value interface{} // 参数值
    Type  string      // 参数类型
}
```

### Fault

表示TR069故障信息。

```go
type Fault struct {
    Code   int    // 故障代码
    String string // 故障描述
}
```

## 使用示例

### 解析Inform消息

```go
// 示例TR-069 Inform消息
xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope
    xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
    <soap-env:Header>
        <cwmp:ID soap-env:mustUnderstand="1">1234567890</cwmp:ID>
    </soap-env:Header>
    <soap-env:Body>
        <cwmp:Inform>
            <DeviceId>
                <Manufacturer>ExampleCorp</Manufacturer>
                <OUI>001122</OUI>
                <ProductClass>ExampleModel</ProductClass>
                <SerialNumber>1234567890</SerialNumber>
            </DeviceId>
            <Event soap-enc:arrayType="cwmp:EventStruct[1]">
                <EventStruct>
                    <EventCode>0 BOOTSTRAP</EventCode>
                    <CommandKey></CommandKey>
                </EventStruct>
            </Event>
            <MaxEnvelopes>1</MaxEnvelopes>
            <CurrentTime>2023-01-01T00:00:00Z</CurrentTime>
            <RetryCount>0</RetryCount>
            <ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[2]">
                <ParameterValueStruct>
                    <n>Device.DeviceInfo.Manufacturer</n>
                    <Value xsi:type="xsd:string">ExampleCorp</Value>
                </ParameterValueStruct>
                <ParameterValueStruct>
                    <n>Device.DeviceInfo.ModelName</n>
                    <Value xsi:type="xsd:string">Model-123</Value>
                </ParameterValueStruct>
            </ParameterList>
        </cwmp:Inform>
    </soap-env:Body>
</soap-env:Envelope>`)

// 创建解析器
parser := factory.NewParser()

// 解析消息
msg, err := parser.ParseMessage(context.Background(), xmlData)
if err != nil {
    log.Fatalf("解析消息失败: %v", err)
}

// 打印解析结果
fmt.Printf("消息方法: %s\n", msg.Method)
fmt.Printf("参数数量: %d\n", len(msg.Parameters))

// 打印参数
for _, param := range msg.Parameters {
    fmt.Printf("参数: %s = %v (%s)\n", param.Name, param.Value, param.Type)
}
```

### 构建响应消息

```go
// 创建构建器
builder := factory.NewBuilder()

// 构建InformResponse消息
params := map[string]interface{}{
    "MaxEnvelopes": 1,
}
response, err := builder.BuildRPCResponse(context.Background(), "InformResponse", params)
if err != nil {
    log.Fatalf("构建响应失败: %v", err)
}

fmt.Printf("响应XML:\n%s\n", response)
```

### 构建故障消息

```go
// 构建故障响应
faultData, err := builder.BuildFault(context.Background(), 9001, "无效参数")
if err != nil {
    log.Fatalf("构建故障消息失败: %v", err)
}

fmt.Printf("故障XML:\n%s\n", faultData)
```

## 扩展功能

TR069协议基础库提供了多种扩展功能，详情请参考[扩展点功能汇总](extension_points_summary.md)文档。

主要扩展功能包括：
- 状态监控
- 动态配置更新
- 配置持久化
- 配置版本管理
- 详细日志记录
- 消息追踪
- 性能分析工具

这些扩展功能可以根据需要选择性使用，以满足不同场景下的需求。