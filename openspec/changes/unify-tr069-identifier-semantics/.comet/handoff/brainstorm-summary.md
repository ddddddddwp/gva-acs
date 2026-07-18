# Brainstorm Summary

- Change: unify-tr069-identifier-semantics
- Date: 2026-07-18
- Status: 已确认

## 确认的技术方案

推荐采用“停机 + 原子数据库切换 + GVA/core 同步破坏性改名”方案：

1. GVA 保留 `command_id` 作为内部主键和 API 不透明参数，命令列表删除 Command ID 搜索框与列，详情、XML、成功提示和复制入口不渲染 Command ID。
2. 命令模型新增 `CWMPID/cwmp_id/cwmpId`。core 构造出站 RPC 后通过 `MarkSending(commandID, cwmpID, sentAt)` 回写；命令发送前保持为空。
3. XML 模型继续通过内部 Command ID 关联命令，但 DTO 只返回报文实际 `cwmpId`，停止读写和返回 XML Request ID。
4. Reboot、Download、Upload 的 CommandKey 由规范 UUID Command ID 删除连字符得到 32 个小写十六进制字符；重试创建新的 Command ID 和 CommandKey，历史值不改写。
5. 外部 `X-Request-ID` 在 GVA/core 入口映射为单次 HTTP 请求的 Trace ID。Trace ID 只进入上下文和日志；Session ID 使用会话 token 或独立生成值，不再复用 Trace ID。
6. core 直接将 `Request.ID` 改为 `Request.TraceID`、`Attributes.RequestID` 改为 `Attributes.TraceID`、`CommandContext.RequestID` 改为 `CommandContext.CWMPID`；不保留废弃别名、规范化桥或兼容周期，GVA 与 core 在同一次变更中完成编译切换。
7. 修改前停止 GVA。数据库一次性将 `tr069_commands.request_id` 重命名为 `cwmp_id`，删除 `tr069_command_xmls.request_id` 及对应索引，然后使用新模型启动；不双写、不保留旧列、不提供旧版本回滚路径。

### 数据流

```text
提交命令
  -> Command ID（内部主键）
  -> 可选派生 CommandKey
  -> QUEUED / WAITING_DEVICE
  -> core 构造 SOAP 并生成 CWMP ID
  -> MarkSending(commandID, cwmpID)
  -> XML sink 保存 commandID + 报文实际 cwmpID
  -> 同步响应通过 CWMP ID 关联
  -> 异步 Inform/TransferComplete 通过 CommandKey 关联
```

```text
X-Request-ID / 自动 UUID
  -> Trace ID（单次 HTTP 请求、日志）

CWMP cookie/session token
  -> Session ID（整个 CWMP 会话）

SOAP <cwmp:ID>
  -> CWMP ID（一次请求/响应）
```

## 已放弃的备选方案

1. 删除并重建 TR-069 命令表：实现最简单，但会丢失全部命令、事件和 XML 历史，不推荐。
2. 渐进双字段迁移：可以兼容旧二进制，但会引入用户明确不需要的过渡代码、双写和清理阶段，不采用。

## 关键取舍与风险

- Command ID 隐藏仅是 UI 简化，不是安全控制；所有详情和重试接口继续依赖 GVA 权限。
- CommandKey 可反推出 Command ID，这是用户明确接受的确定性派生权衡。
- 原子 DDL 重命名保留现有命令数据，但不会兼容旧程序；切换后旧二进制不得再次启动。
- core 是未上线 SDK，因此直接删除旧字段；GVA 和 core 必须在同一开发窗口完成并一起验证。
- Session ID 从首次 Trace ID 解耦后，旧持久化 Session 可能保留历史值，但新建会话必须使用独立 Session ID；不需要批量迁移短生命周期 Session。
- 数据库切换失败时保持 GVA 停止，修复 DDL 后再启动；不设计应用级回滚桥。

## 测试策略

- core：先写失败测试，覆盖 TraceID 传播、Session/Trace 分离、MarkSending 的 CWMP ID 语义，并用编译检查确认旧字段已完全移除。
- GVA：SQLite schema cutover 测试覆盖列重命名、XML 旧列删除、重复执行安全和现有命令数据保留；仓储测试覆盖 `cwmp_id` 写入与查询。
- CommandKey：覆盖精确 32 字符、十六进制、确定性、仅三种 RPC 生成、重试新值、历史值不改写及异步关联。
- XML/API：覆盖 DTO 不再出现 `requestId`、XML 保留实际 `cwmpId`、Trace ID 不持久化。
- 前端：contract tests 覆盖删除搜索框/列/详情字段/成功提示中的 Command ID，内部行键和接口参数仍可用，CommandKey 条件展示。
- 运行：core 与 GVA 全量测试、前端 contract 与 build，通过后重启 GVA 前后端并检查 18888、18080、7458 服务健康。

## Spec Patch

已回写 OpenSpec：删除渐进迁移和 core 一版兼容要求，改为停机一次数据库切换及 GVA/core 同步破坏性改名。
