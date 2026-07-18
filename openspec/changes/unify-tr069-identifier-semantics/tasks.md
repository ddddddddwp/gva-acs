## 1. tr069-core 标识语义

- [x] 1.1 为 `Request.TraceID` 和可观测 `TraceID` 传播编写失败测试，并用编译检查约束旧 Request ID 字段已删除
- [x] 1.2 直接将 core 请求与可观测属性切换到 Trace ID 新字段，删除旧字段和兼容逻辑
- [x] 1.3 为同一 CWMP Session 的多次 HTTP 请求编写 Trace ID 与 Session ID 分离测试并修正会话日志
- [x] 1.4 将 `CommandContext.RequestID` 改为 `CWMPID`，将 `MarkSending` 参数语义改为 `cwmpID` 并通过 core 全量测试

## 2. GVA 数据模型与迁移

- [x] 2.1 为停机执行的 `tr069_commands.request_id` 到 `cwmp_id` 列重命名及 XML 旧列删除编写数据库结构测试
- [x] 2.2 将 GVA 命令模型、仓储、状态转换和查询关联切换到 `CWMPID/cwmp_id/cwmpId`
- [x] 2.3 停止读写 XML Request ID，更新 XML sink、模型和 DTO，使其只返回报文实际 CWMP ID
- [x] 2.4 更新 GVA 命令 API、服务和存储测试，确认新代码不依赖 `requestId`

## 3. CommandKey 派生与异步关联

- [x] 3.1 为规范 UUID 派生 32 字符小写十六进制 CommandKey 编写失败测试
- [x] 3.2 为 Reboot、Download、Upload 实现由 Command ID 确定性派生 CommandKey，普通 RPC 保持为空
- [x] 3.3 验证重试生成新的 Command ID 与 CommandKey、`retry_of` 关联正确且历史 CommandKey 不被改写
- [x] 3.4 验证 Reboot Inform 与 Download/Upload TransferComplete 继续通过 CommandKey 幂等关联原命令

## 4. GVA Trace 与用户界面

- [x] 4.1 为 GVA HTTP Trace 中间件编写测试，确认每次请求独立生成内部 `TraceID/traceId` 且不读取或回写外部链路头
- [x] 4.2 将 GVA Trace 上下文、调试路由和结构化日志字段改为 Trace ID，并确保 Trace ID 不进入命令/XML DTO
- [x] 4.3 编写前端契约测试，要求列表、详情、XML 和成功提示均不渲染 Command ID 或 Request ID
- [x] 4.4 更新 RPC 命令 UI：展示 CWMP ID，按状态显示未生成文案，仅对 Reboot、Download、Upload 展示 CommandKey

## 5. 验证与运行检查

- [x] 5.1 运行 tr069-core 全量 Go 测试并确认旧 Request ID 字段和调用方式已完全移除
- [x] 5.2 运行 GVA TR-069 插件全量 Go 测试、数据库结构切换测试和前端 contract tests
- [x] 5.3 构建 GVA 前端与后端，确认 API JSON 中用户可见字段只使用 `cwmpId` 和 `traceId`
- [x] 5.4 停止 GVA、执行数据库一次性结构切换，再重启前后端并验证命令详情、XML 详情及服务健康状态

## 6. 审查修复

- [x] 6.1 为使用新 CWMP ID 的 Reboot Inform 和 TransferComplete 补充失败测试，并通过 CommandKey 或设备启动事件关联原命令，同时保留报文实际 CWMP ID
- [x] 6.2 为 core HTTP 适配器补充边界测试，确认 Trace ID 仅在内部生成且不通过 HTTP 响应暴露
- [x] 6.3 为 core 命令发送上下文补充失败测试，并将实际 SOAP CWMP ID 写入 `CommandContext.CWMPID`
- [x] 6.4 运行 tr069-core、GVA TR-069 插件全量测试及相关构建，并使用 BS 验证 ACS 响应不再包含额外链路头
