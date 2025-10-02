# TR069 会话管理模块

本模块实现了TR069协议的会话管理功能，提供了会话的创建、获取、更新、关闭等基本操作，以及会话状态监控、超时清理等高级功能。

## 主要功能

- 会话生命周期管理（创建、更新、关闭）
- 会话状态跟踪和监控
- 会话超时和自动清理
- 基于设备ID的会话分组
- 会话事件通知机制

## 使用示例

```go
// 创建会话管理器
sessionManager := session.NewSessionManager()

// 设置会话超时时间
sessionManager.SetSessionTimeout(15 * time.Minute)

// 创建新会话
sessionInfo, err := sessionManager.CreateSession(ctx, "device123", 
    interfaces.WithSessionMetadata("clientIP", "192.168.1.100"))
if err != nil {
    log.Fatalf("Failed to create session: %v", err)
}

// 获取会话信息
session, err := sessionManager.GetSession(ctx, sessionInfo.ID)
if err != nil {
    log.Fatalf("Failed to get session: %v", err)
}

// 更新会话
updatedSession, err := sessionManager.UpdateSession(ctx, sessionInfo.ID,
    interfaces.WithSessionMetadata("lastCommand", "GetParameterValues"))
if err != nil {
    log.Fatalf("Failed to update session: %v", err)
}

// 关闭会话
err = sessionManager.CloseSession(ctx, sessionInfo.ID)
if err != nil {
    log.Fatalf("Failed to close session: %v", err)
}

// 注册会话事件监听器
sessionManager.RegisterSessionListener(&MySessionListener{})
```

## 性能考虑

- 使用读写锁保证并发安全
- 定期清理过期会话，避免内存泄漏
- 事件通知使用异步方式，避免阻塞主流程

## 扩展点

- 可通过实现`SessionEventListener`接口来处理会话事件
- 可通过添加新的`SessionOption`函数来扩展会话配置选项
- 可以通过继承`sessionManager`结构体来添加自定义功能