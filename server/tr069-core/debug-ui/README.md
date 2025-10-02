# TR069 Debug UI

这是一个独立的TR069协议调试工具，与TR069协议库完全解耦，提供直观的Web界面用于TR069消息的解析、构建和调试。

## 项目结构

```
debug-ui/
├── frontend/           # 前端静态文件
│   ├── css/           # 样式文件
│   ├── js/            # JavaScript文件
│   ├── html/          # HTML模板
│   ├── index.html     # 主页面
│   └── sessions.html  # 会话管理页面
├── backend/           # 后端服务器代码
│   └── main.go        # HTTP API服务器
├── api/              # API接口定义
│   └── types.go       # API数据类型
├── go.mod            # 独立的Go模块
└── README.md         # 本文档
```

## 设计原则

1. **完全独立**: 调试工具与TR069协议库完全分离，不直接依赖协议库代码
2. **HTTP API通信**: 通过RESTful API与TR069协议库进行通信
3. **调试专用**: 仅用于开发和调试，不包含在生产环境的协议库中
4. **现代化UI**: 基于Bootstrap 5的响应式Web界面
5. **实时交互**: JavaScript实现的动态用户体验

## 快速开始

### 启动服务器

```bash
cd debug-ui
go run backend/main.go
```

服务器将在 `http://localhost:8080` 启动。

### 访问界面

打开浏览器访问 `http://localhost:8080` 即可使用调试工具。

## 核心功能

### 1. TR069消息解析
- **XML消息解析**: 将TR069 SOAP消息解析为结构化数据
- **参数提取**: 自动提取消息中的方法、ID、参数等信息
- **格式验证**: 验证消息格式的正确性
- **实时预览**: 即时显示解析结果

### 2. TR069消息构建
- **方法选择**: 支持常见的TR069方法（GetParameterValues、SetParameterValues等）
- **参数编辑**: 可视化的参数编辑器
- **XML生成**: 自动生成符合TR069标准的SOAP消息
- **格式化输出**: 美化的XML输出格式

### 3. 服务器状态监控
- **在线状态**: 实时显示服务器运行状态
- **运行时间**: 显示服务器启动时间和运行时长
- **版本信息**: 显示调试工具版本信息

## API接口

### 解析接口
```
POST /api/parse
Content-Type: application/json

{
  "message": "<?xml version=\"1.0\"?>..."
}
```

### 构建接口
```
POST /api/build
Content-Type: application/json

{
  "method": "GetParameterValues",
  "parameters": {
    "ParameterNames": ["Device.DeviceInfo.SerialNumber"]
  }
}
```

### 状态接口
```
GET /api/status
```

## 技术栈

### 后端
- **Go**: 高性能的HTTP服务器
- **Gorilla Mux**: HTTP路由器
- **标准库**: 使用Go标准库实现核心功能

### 前端
- **Bootstrap 5**: 现代化的CSS框架
- **Vanilla JavaScript**: 原生JavaScript实现
- **响应式设计**: 支持桌面和移动设备

## 开发指南

### 添加新的TR069方法

1. 在后端的 `handleBuild` 函数中添加新方法的处理逻辑
2. 在前端的 `buildMessage` 函数中添加对应的参数处理
3. 更新UI中的方法选择器

### 扩展解析功能

1. 修改 `handleParse` 函数中的解析逻辑
2. 更新 `ParseResponse` 结构体以支持新的字段
3. 在前端更新解析结果的显示逻辑

## 部署说明

### 开发环境
```bash
# 启动开发服务器
go run backend/main.go
```

### 生产环境
```bash
# 编译二进制文件
go build -o debug-ui backend/main.go

# 运行
./debug-ui
```

## 注意事项

1. **仅用于调试**: 此工具仅用于开发和调试，不应在生产环境中使用
2. **安全考虑**: 工具不包含身份验证，请在安全的网络环境中使用
3. **性能限制**: 为简化实现，某些功能可能有性能限制
4. **兼容性**: 支持TR069 1.0标准，部分扩展功能可能需要额外配置

## 故障排除

### 常见问题

1. **端口占用**: 如果8080端口被占用，请修改 `backend/main.go` 中的端口配置
2. **静态文件404**: 确保 `frontend` 目录存在且包含所有必要文件
3. **API调用失败**: 检查后端服务器是否正常启动

### 日志查看

服务器启动后会在控制台输出详细的日志信息，包括：
- 服务器启动状态
- API请求日志
- 错误信息

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

本项目遵循与主TR069协议库相同的许可证。