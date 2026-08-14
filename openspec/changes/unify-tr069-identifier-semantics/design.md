## Context

GVA 在命令提交时生成 UUID 形式的 Command ID，并用它关联命令、事件、XML、队列和重试。tr069-core 在实际构造出站 RPC 时生成 SOAP `<cwmp:ID>`，再通过 `MarkSending` 回写 GVA；当前回写字段名为 Request ID，导致协议 ID 与 HTTP 链路 ID 混用。XML 记录已经保存 CWMP ID，但还保留一个通常为空的 Request ID。前端因此同时展示 Command ID、Request ID、CWMP ID 和 CommandKey，用户难以判断各字段用途。

core 的 HTTP Request ID 同时被用作可观测属性和 Session ID 初始值，使一次 HTTP 请求的 Trace 生命周期与一次 CWMP 会话的 Session 生命周期耦合。现有 Reboot、Download、Upload CommandKey 使用 `rpc-<UUID>`，总长 40 字符，超过 TR-069 `string(32)` 的限制。

本变更影响 GVA 数据库、后端 DTO 和仓储、前端命令记录以及独立维护的 tr069-core SDK。GVA 权限系统继续作为访问控制边界；任何标识值本身都不是授权凭证。

## Goals / Non-Goals

**Goals:**

- 为 Command ID、CWMP ID、CommandKey、Trace ID 和 Session ID 建立唯一、稳定且可测试的语义。
- 保留 Command ID 作为内部生命周期主键，同时从普通 UI 中移除。
- 让命令表和 XML 记录明确使用 `cwmp_id/CWMPID/cwmpId` 表示 SOAP ID。
- 生成符合 `string(32)` 约束的确定性 CommandKey，并保持异步结果关联可靠。
- 让 Trace ID 只追踪单次 HTTP 请求，让 Session ID 独立追踪 CWMP 会话。
- 保持 GVA 详情、重试等接口受现有权限控制，并完成数据库、SDK、API 和前端契约测试。

**Non-Goals:**

- 不使用 CWMP ID 取代 Command ID 作为数据库主键。
- 不新增 Public ID、签名令牌或另一套面向用户的命令标识。
- 不把 Command ID、CWMP ID 或 CommandKey 当作权限凭证。
- 不重构 RPC 状态机、设备队列、超时策略或权限模型。
- 不批量重写历史 CommandKey，也不让 Trace ID 进入命令详情或 XML 持久化。

## Decisions

### 1. Command ID 保持内部主键，但退出普通 UI

`tr069_commands.command_id` 继续作为主键，命令事件、XML、重试来源、队列和后台扫描器继续通过它关联。命令列表、命令详情、XML 标签和成功提示不渲染 Command ID。前端仍可把 API 返回的 Command ID 当作不透明行键和详情/重试参数，后端继续执行 GVA 权限检查。

选择该方案而不是 CWMP ID 主键，是因为命令在排队、等待设备和构造失败阶段已经需要稳定身份，而此时可能尚未产生任何 SOAP 报文。选择该方案而不是新增 Public ID，是为了避免第五套标识及额外映射成本。

### 2. CWMP ID 只表示一次 SOAP 请求/响应关联

命令模型新增明确的 `CWMPID string`，持久化列和 JSON 字段分别为 `cwmp_id`、`cwmpId`。core 构造出站请求后调用 `MarkSending(commandID, cwmpID, sentAt)` 回写。响应使用相同 CWMP ID 关联该次协议交换。

命令尚未发送时 CWMP ID 为空；UI 根据状态显示“尚未生成”或“未生成（构造失败）”。XML 记录保存每条报文实际携带的 CWMP ID，因此 TransferComplete、Inform 等设备主动报文可以拥有不同于原命令请求的 CWMP ID。

设备主动异步报文不得依赖原请求 CWMP ID 反查命令：TransferComplete 优先通过 CommandKey 关联 Download/Upload，携带 CommandKey 的 Inform 同样通过 CommandKey 关联；未携带 CommandKey 的 Reboot 启动 Inform 则通过设备标识、启动事件和等待重启状态关联。完成关联后，XML 仍保存设备报文自身的 CWMP ID。

### 3. CommandKey 由 Command ID 确定性派生

仅 Reboot、Download、Upload 在创建命令时生成 CommandKey。算法为：解析规范 UUID Command ID，将规范字符串中的四个连字符删除，得到 32 个小写十六进制字符。

```text
Command ID:  62a53a00-786f-4a45-b31f-b23f7d23bb6e
CommandKey:  62a53a00786f4a45b31fb23f7d23bb6e
```

重试创建新的 Command ID，因此自然得到新的 CommandKey，并通过 `retry_of` 关联原命令。历史 CommandKey 原样保留。完整 CommandKey 可以反推出 Command ID，这被接受为既定权衡；Command ID 只是弱化展示，不是秘密。

### 4. Trace ID 与 Session ID 完全分离

Trace ID 表示一次 HTTP 请求，只存在于请求上下文和结构化日志。GVA 与 core 内部统一使用 `TraceID/traceId`，并在每个 HTTP 请求入口生成 UUID。ACS 不读取也不回写外部链路标识头，避免非标准响应头触发 CPE 厂商协议栈兼容问题。

Session ID 表示一次 CWMP 会话，独立创建和持久化。core 不再用 HTTP Trace ID 初始化 Session ID，也不再把 Session ID 写入 `trace_id` 日志字段。结构化日志可以同时记录 `traceId`、`sessionId`、`cwmpId`、`commandId`，但 RPC 命令详情不显示 Trace ID。

### 5. 数据库采用停机一次切换

修改前停止 GVA。数据库直接将 `tr069_commands.request_id` 重命名为 `cwmp_id` 并保留索引，删除 `tr069_command_xmls.request_id` 及其索引，然后只允许新版本 GVA 连接。应用不双写、不保留旧列，也不提供旧程序兼容路径。

结构切换不修改 Command ID、CommandKey、状态、事件或 XML payload。DDL 失败时 GVA 保持停止，修复数据库结构后再启动新版本。切换完成后旧版本 GVA 不得再次连接该数据库。

### 6. core SDK 直接切换到新语义

core 直接使用 `Request.TraceID`、`observability.Attributes.TraceID` 和 `CommandContext.CWMPID`，删除公开的 `Request.ID`、`Attributes.RequestID` 和 `CommandContext.RequestID`。不保留废弃别名、入口规范化桥或兼容周期。GVA 与 core 在同一个开发窗口完成修改并一起通过编译和测试。

`CommandRepo.MarkSending` 只修改参数名称 `requestID` 为 `cwmpID`，Go 接口的方法类型不因参数名变化而改变。

### 7. UI 以协议和状态为中心

命令详情展示功能、设备、状态、截止时间、CWMP ID、时间字段和重试来源语义；普通 RPC 不展示 CommandKey，Reboot、Download、Upload 才展示。完整 XML 只展示方向、方法、时间、CWMP ID 和 XML payload。Command ID 继续存在于前端数据对象中，但不提供文本、列、提示或复制入口。

## Risks / Trade-offs

- [隐藏 UI 不等于隐藏 API] → 文档明确 Command ID 是不透明内部参数而非秘密，所有接口继续依赖 GVA 权限验证。
- [CommandKey 可反推 Command ID] → 接受该确定性派生权衡，不把两者作为授权凭证。
- [停机 DDL 失败导致服务不可用] → 切换期间保持 GVA 停止，数据库达到新结构且验证通过后才启动服务。
- [Trace 与 Session 拆分改变日志关联方式] → 日志同时写入明确命名的 `traceId` 和 `sessionId`，增加覆盖连续 POST 的测试。
- [core 旧调用者无法编译] → SDK 尚未上线，GVA 和 core 同步修改，不保留旧调用方式。
- [历史 40 字符 CommandKey 仍存在] → 只要求新命令符合规范，历史记录保持审计真实性和异步关联能力。

## Migration Plan

1. 在 core 中直接改用 Trace ID、Session ID 和 CWMP ID 新语义，删除旧字段并更新所有调用者和测试。
2. 将 GVA 模型、仓储、关联、DTO、Trace 中间件和日志切换到新字段。
3. 将新 CommandKey 生成切换为 32 字符确定性派生，并验证重试与异步关联。
4. 更新前端展示契约，移除 Command ID 和 Request ID 的可见元素。
5. 停止 GVA 前后端，一次性重命名命令表列并删除 XML 旧列。
6. 运行 core、GVA 插件、数据库结构和前端契约测试。
7. 重新启动 GVA 前后端，验证命令详情和 XML 展示。

## Open Questions

无。CommandKey 可反推 Command ID、Command ID 仅弱化展示、数据库停机切换以及 core 不保留兼容字段均已由用户确认。
