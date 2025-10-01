# TR069 Core Library

A high-performance, modular TR069 protocol parsing and building library for Go.

[![Go Report Card](https://goreportcard.com/badge/github.com/example/tr069-core)](https://goreportcard.com/report/github.com/example/tr069-core)
[![GoDoc](https://godoc.org/github.com/example/tr069-core?status.svg)](https://godoc.org/github.com/example/tr069-core)
[![License](https://img.shields.io/github/license/example/tr069-core.svg)](LICENSE)

## 目录

- [概述](#概述)
- [特性](#特性)
- [架构](#架构)
- [安装](#安装)
- [快速开始](#快速开始)
- [核心组件](#核心组件)
- [调试工具](#调试工具)
- [扩展功能](#扩展功能)
- [性能指标](#性能指标)
- [文档](#文档)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

## 概述

TR069 Core Library 是一个高性能、模块化的 TR069 协议解析与构建基础库，专为需要与 TR069 设备通信的应用程序设计。本库提供了完整的 TR069 协议实现，专注于性能、模块化和易用性，包括解析和构建 TR069 消息、处理事件、流式处理大型 XML 消息以及缓存频繁访问的数据等功能。

## 特性

- **高性能**：针对速度和低内存使用进行优化，支持零内存分配模式
- **模块化设计**：明确定义的接口和关注点分离，便于扩展和维护
- **接口驱动**：所有功能通过接口暴露，实现与接口分离
- **事件通知**：完整的事件通知机制，支持订阅和发布
- **流处理**：高效处理大型 XML 消息，最小化内存使用
- **智能缓存**：支持多种淘汰策略的缓存系统
- **线程安全**：设计用于并发环境
- **可扩展性**：易于扩展自定义实现和功能
- **RESTful API**：提供 HTTP/JSON 格式的 RESTful 接口封装

## 架构

TR069 Core Library 采用分层架构设计：

```
应用层 → 接口层 → 实现层 → 基础组件层
```

- **接口层**：定义所有公开 API 和数据类型
- **实现层**：提供接口的具体实现
- **基础组件层**：提供通用功能和工具

## 安装

```bash
go get github.com/example/tr069-core
```

## 快速开始

### 基本解析器使用

```go
import "github.com/example/tr069-core/interfaces"

// 创建解析器
parser := tr069.NewParser()

// 解析 TR069 消息
message, err := parser.ParseMessage(xmlData)
if err != nil {
    // 处理错误
}

// 处理消息
fmt.Println("RPC 方法:", message.GetRPCMethod())
```

### 基本构建器使用

```go
import "github.com/example/tr069-core/interfaces"

// 创建构建器
builder := tr069.NewBuilder()

// 构建 TR069 消息
xmlData, err := builder.BuildMessage(message)
if err != nil {
    // 处理错误
}

// 使用生成的 XML 数据
fmt.Println(string(xmlData))
```

### RESTful API 使用

```go
import "github.com/example/tr069-core/pkg/api"

// 创建 API 服务器
server := api.NewServer(api.APIConfig{
    Port: 8080,
    BasePath: "/api/v1",
})

// 注册处理器
server.RegisterHandler("/devices", deviceHandler)

// 启动服务器
err := server.Start()
if err != nil {
    log.Fatalf("无法启动服务器: %v", err)
}
```

## 核心组件

### 解析器 (Parser)

处理从 XML 格式解析 TR069 协议消息。支持严格和非严格模式，以实现灵活解析。

### 构建器 (Builder)

从内部表示构造 XML 格式的 TR069 协议消息。

### 事件通知 (Event Notification)

提供完整的事件通知系统：
- 事件发布和订阅
- 可配置限制的事件队列
- 可靠传递的重试策略

### 流解析器 (Stream Parser)

使用流式技术高效处理大型 XML 消息，最小化内存使用。

### 缓存 (Cache)

支持多种淘汰策略的智能缓存系统：
- LRU（最近最少使用）
- LFU（最不经常使用）
- FIFO（先进先出）

### RESTful API

提供 HTTP/JSON 格式的 RESTful 接口，封装 TR069 功能：
- 设备管理 API
- 参数管理 API
- RPC 调用 API
- 事件订阅 API

## 调试工具

TR069 Core Library 提供了一个独立的调试工具，用于开发和测试 TR069 协议消息。

### 特性

- **独立运行**：调试工具与核心库完全解耦，可独立部署和运行
- **Web 界面**：现代化的 Bootstrap 5 响应式界面
- **实时解析**：即时解析 TR069 XML 消息并显示结构化结果
- **消息构建**：可视化构建 TR069 协议消息
- **API 接口**：提供 RESTful API 用于程序化调用

### 快速启动

```bash
# 进入调试工具目录
cd debug-ui

# 启动调试服务器
go run backend/main.go

# 访问 Web 界面
# 打开浏览器访问 http://localhost:8080
```

### 主要功能

1. **消息解析**
   - 解析 TR069 SOAP 消息
   - 提取方法、参数和元数据
   - 验证消息格式

2. **消息构建**
   - 支持常见 TR069 方法
   - 可视化参数编辑
   - 生成标准 SOAP 消息

3. **服务器监控**
   - 实时状态显示
   - 运行时间统计
   - 版本信息查看

### API 端点

```bash
# 解析消息
POST /api/parse
Content-Type: application/json
{
  "message": "<?xml version=\"1.0\"?>..."
}

# 构建消息
POST /api/build
Content-Type: application/json
{
  "method": "GetParameterValues",
  "parameters": {...}
}

# 查看状态
GET /api/status
```

详细使用说明请参考 [调试工具文档](debug-ui/README.md)。

## 扩展功能

TR069 Core Library 提供多个扩展点，详见 [扩展点文档](docs/extension_points.md)：

- 安全性增强功能
- 协议扩展支持
- 性能优化功能
- 管理功能增强
- 高级配置功能
- 调试和诊断功能
- RESTful API 封装

## 性能指标

- 单消息解析时间: < 100μs
- 内存分配次数: 每消息 ≤ 5次
- 并发处理: 支持1000+并发解析
- 吞吐量: > 10,000 msg/s (单核)

## 文档

- [接口使用文档](docs/interface_usage.md)
- [扩展点文档](docs/extension_points.md)
- [安全增强文档](docs/security_enhancements.md)
- [API 文档](docs/api_documentation.md)

## 贡献指南

欢迎贡献代码、报告问题或提出功能请求。请遵循以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件