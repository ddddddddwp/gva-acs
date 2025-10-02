# TR069协议基础库集成指南

本文档提供详细指导，说明如何在其他项目中引入和使用TR069协议基础库的核心接口层功能。

## 目录

1. [引入库](#引入库)
   - [使用go mod](#使用go-mod)
   - [直接复制代码](#直接复制代码)
2. [初始化](#初始化)
   - [创建解析器和构建器](#创建解析器和构建器)
   - [配置选项](#配置选项)
3. [基本使用场景](#基本使用场景)
   - [解析TR069消息](#解析tr069消息)
   - [构建TR069响应](#构建tr069响应)
   - [处理故障情况](#处理故障情况)
4. [高级使用场景](#高级使用场景)
   - [自定义配置](#自定义配置)
   - [并发处理](#并发处理)
   - [性能优化](#性能优化)
5. [最佳实践](#最佳实践)
   - [错误处理](#错误处理)
   - [内存管理](#内存管理)
   - [日志记录](#日志记录)
6. [常见问题](#常见问题)

## 引入库

### 使用go mod

在你的项目中引入TR069协议基础库的最简单方法是使用Go模块系统。

1. 确保你的项目已初始化为Go模块：

```bash
cd your-project
go mod init github.com/your-username/your-project
```

2. 添加TR069协议基础库依赖：

```bash
go get github.com/your-organization/tr069
```

3. 在代码中导入需要的包：

```go
import (
    "github.com/your-organization/tr069/factory"
    "github.com/your-organization/tr069/interface"
)
```

### 直接复制代码

如果你需要对库进行深度定制，也可以直接复制源代码到你的项目中：

1. 复制核心代码目录：

```bash
cp -r /path/to/tr069/{factory,interface,internal} /path/to/your-project/tr069/
```

2. 调整导入路径以匹配你的项目结构。

## 初始化

### 创建解析器和构建器

在使用TR069协议基础库前，需要创建解析器和构建器实例：

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/your-organization/tr069/factory"
)

func main() {
    // 创建默认解析器
    parser := factory.NewParser()
    
    // 创建默认构建器
    builder := factory.NewBuilder()
    
    // 现在可以使用parser和builder进行TR069消息的解析和构建
}
```

### 配置选项

TR069协议基础库支持通过选项模式进行配置：

```go
// 创建自定义配置的解析器
parser := factory.NewParser(
    factory.WithStrictMode(true),    // 启用严格模式，更严格的XML验证
    factory.WithMaxDepth(100),       // 设置最大解析深度
)

// 创建自定义配置的构建器
builder := factory.NewBuilder(
    factory.WithPrettyPrint(true),   // 启用美化输出
    factory.WithNamespace("urn:dslforum-org:cwmp-1-0"), // 设置命名空间
)
```

## 基本使用场景

### 解析TR069消息

以下是解析TR069 Inform消息的示例：

```go
// 假设xmlData包含从CPE接收到的TR069 Inform消息
xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope
    xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
    <soap-env:Header>
        <cwmp:ID soap-env:mustUnderstand="1">1234567890</cwmp:ID>
    </soap-env:Header>
    <soap-env:Body>
        <cwmp:Inform>
            <!-- Inform消息内容 -->
        </cwmp:Inform>
    </soap-env:Body>
</soap-env:Envelope>`)

// 创建上下文
ctx := context.Background()

// 解析消息
msg, err := parser.ParseMessage(ctx, xmlData)
if err != nil {
    log.Fatalf("解析消息失败: %v", err)
}

// 获取消息ID
fmt.Printf("消息ID: %s\n", msg.ID)

// 获取RPC方法名
fmt.Printf("RPC方法: %s\n", msg.Method)

// 访问参数
for _, param := range msg.Parameters {
    fmt.Printf("参数名: %s, 值: %v, 类型: %s\n", param.Name, param.Value, param.Type)
}
```

### 构建TR069响应

以下是构建TR069 InformResponse消息的示例：

```go
// 创建响应消息
response, err := builder.BuildRPCResponse(ctx, "InformResponse", map[string]interface{}{
    "MaxEnvelopes": 1,
})
if err != nil {
    log.Fatalf("构建响应失败: %v", err)
}

// 输出响应XML
fmt.Println(string(response))
```

### 处理故障情况

当需要返回故障信息时：

```go
// 构建故障响应
faultResponse, err := builder.BuildFault(ctx, 9005, "Invalid Parameter Name")
if err != nil {
    log.Fatalf("构建故障响应失败: %v", err)
}

// 输出故障响应XML
fmt.Println(string(faultResponse))
```

## 高级使用场景

### 自定义配置

TR069协议基础库支持多种自定义配置，以适应不同的使用场景：

```go
// 创建自定义配置
config := factory.NewConfig(
    factory.WithTimeout(5 * time.Second),
    factory.WithRetryCount(3),
    factory.WithLogLevel("debug"),
)

// 使用自定义配置创建解析器和构建器
parser := factory.NewParserWithConfig(config)
builder := factory.NewBuilderWithConfig(config)
```

### 并发处理

TR069协议基础库设计为线程安全，可以在并发环境中使用：

```go
// 创建一个工作池处理多个TR069消息
func processMessages(messages [][]byte) {
    parser := factory.NewParser()
    
    var wg sync.WaitGroup
    for _, msgData := range messages {
        wg.Add(1)
        go func(data []byte) {
            defer wg.Done()
            
            ctx := context.Background()
            msg, err := parser.ParseMessage(ctx, data)
            if err != nil {
                log.Printf("解析消息失败: %v", err)
                return
            }
            
            // 处理消息
            processMessage(msg)
        }(msgData)
    }
    
    wg.Wait()
}
```

### 性能优化

对于高性能场景，可以使用对象池减少内存分配：

```go
// 使用对象池
pool := factory.NewMessagePool(100) // 预分配100个消息对象

// 从池中获取消息对象
msg := pool.Get()

// 使用完毕后归还对象
pool.Put(msg)
```

## 最佳实践

### 错误处理

始终检查返回的错误，并根据错误类型采取适当的处理措施：

```go
msg, err := parser.ParseMessage(ctx, xmlData)
if err != nil {
    switch e := err.(type) {
    case *errors.XMLParseError:
        // 处理XML解析错误
        log.Printf("XML解析错误: %v, 位置: %d", e.Message, e.Position)
    case *errors.ValidationError:
        // 处理验证错误
        log.Printf("验证错误: %v", e.Message)
    default:
        // 处理其他错误
        log.Printf("未知错误: %v", err)
    }
    return
}
```

### 内存管理

对于高频率消息处理，建议使用对象池和预分配缓冲区：

```go
// 预分配缓冲区
buffer := make([]byte, 0, 4096)

// 使用预分配的缓冲区构建消息
response, err := builder.BuildMessageWithBuffer(ctx, msg, buffer)
```

### 日志记录

配置适当的日志级别以便调试：

```go
// 设置日志级别
factory.SetLogLevel("debug") // 可选值: debug, info, warn, error

// 设置自定义日志处理器
factory.SetLogHandler(func(level string, format string, args ...interface{}) {
    // 自定义日志处理逻辑
    yourLogger.Log(level, fmt.Sprintf(format, args...))
})
```

## 常见问题

**Q: 如何处理不同版本的TR069协议?**

A: TR069协议基础库支持通过命名空间设置处理不同版本：

```go
// 设置TR069协议版本
builder := factory.NewBuilder(
    factory.WithNamespace("urn:dslforum-org:cwmp-1-2"), // TR069 v1.2
)
```

**Q: 如何处理大型XML消息以避免内存问题?**

A: 使用流式解析器处理大型消息：

```go
// 创建流式解析器
streamParser := factory.NewStreamParser()

// 设置处理器
streamParser.OnMethod(func(method string) {
    fmt.Printf("检测到方法: %s\n", method)
})

streamParser.OnParameter(func(name string, value interface{}) {
    fmt.Printf("参数: %s = %v\n", name, value)
})

// 流式处理
err := streamParser.Parse(ctx, reader)
if err != nil {
    log.Fatalf("流式解析失败: %v", err)
}
```

**Q: 如何扩展库以支持自定义功能?**

A: 可以通过实现接口或使用扩展点来添加自定义功能：

```go
// 实现自定义解析器
type MyParser struct {
    interface.Parser
    // 自定义字段
}

// 实现接口方法
func (p *MyParser) ParseMessage(ctx context.Context, data []byte) (*interface.Message, error) {
    // 自定义实现
    // ...
    
    // 调用原始实现
    return p.Parser.ParseMessage(ctx, data)
}
```

---

通过本指南，你应该能够在自己的项目中成功集成和使用TR069协议基础库。如有更多问题，请参考项目文档或联系维护团队。