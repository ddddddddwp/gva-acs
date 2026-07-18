## Why

GVA 与 tr069-core 当前混用了 Command ID、CWMP ID 和 Request ID：命令表的 `request_id` 实际保存 CWMP ID，core 的 Request ID 又承担 HTTP 日志链路标识，前端同时展示多个含义重叠或为空的字段。与此同时，现有 `rpc-<UUID>` CommandKey 长达 40 字符，超过 TR-069 `string(32)` 的协议约束，存在设备兼容性风险。

本变更将四类标识的语义、生命周期、持久化和展示边界统一下来，使命令记录、协议关联、异步完成和日志排查各自使用明确且可测试的标识。

## What Changes

- 明确 Command ID 为 GVA 内部命令主键，继续关联命令、事件、XML、队列和重试生命周期，但从普通用户界面中移除。
- 将命令表中误命名的 `request_id` 迁移为 `cwmp_id`，GVA 与 core 统一使用 `CWMPID/cwmpId` 表示 SOAP `<cwmp:ID>`。
- XML 记录只保存并展示报文实际的 CWMP ID，删除空的 Request ID 字段。
- 将 HTTP 和内部日志链路标识统一命名为 Trace ID，并与 Session ID、CWMP ID 分离；Trace ID 不进入 RPC 命令详情。
- Reboot、Download、Upload 的新 CommandKey 由 Command ID 删除连字符后确定性派生为 32 个十六进制字符；重试使用新的 Command ID 和 CommandKey。
- 历史 CommandKey 保持原值，不批量重写已发送、进行中或已完成记录。
- 普通 UI 不展示 Command ID，但前端可继续把它作为受 GVA 权限保护的详情、重试等接口的内部不透明参数。
- 停止 GVA 后一次性将命令表的 `request_id` 重命名为 `cwmp_id` 并删除 XML 表的 `request_id`，不设置双写或兼容阶段。
- **BREAKING**：GVA API 的用户可见命令字段由 `requestId` 改为 `cwmpId`，XML DTO 删除 `requestId`；tr069-core 直接删除旧 Request ID 字段并使用 `TraceID` 和 `CWMPID`。

## Capabilities

### New Capabilities

- `tr069-identifier-semantics`: 规定 GVA 与 tr069-core 中 Command ID、CWMP ID、CommandKey、Trace ID 和 Session ID 的职责、持久化、展示、关联及兼容行为。

### Modified Capabilities

- 无。

## Impact

- GVA 后端：TR-069 命令模型、XML 模型、数据库迁移、命令仓储、命令管理器、Trace 中间件、调试路由和响应 DTO。
- tr069-core SDK：请求模型、可观测属性、HTTP 适配器、命令仓储接口、会话日志和命令上下文。
- GVA 前端：RPC 命令列表、命令详情、XML 元数据和成功提示。
- 数据库：停机执行 `tr069_commands.request_id` 到 `cwmp_id` 的一次性列重命名，并删除 `tr069_command_xmls.request_id`。
- 测试：CommandKey 派生、重试、CWMP 请求响应关联、Trace/Session 分离、数据库迁移、API DTO 和前端展示契约。
