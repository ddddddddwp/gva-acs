# Comet Design Handoff

- Change: unify-tr069-identifier-semantics
- Phase: design
- Mode: compact
- Context hash: fa607740f9f2f88d36866a820b222937352c1390784f93a561ea6a3457c128f2

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/unify-tr069-identifier-semantics/proposal.md

- Source: openspec/changes/unify-tr069-identifier-semantics/proposal.md
- Lines: 1-35
- SHA256: 2912271da1c008ed787cae9e0b63586af12dd713af49344b7f68f5d2753ddba6

```md
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

```

## openspec/changes/unify-tr069-identifier-semantics/design.md

- Source: openspec/changes/unify-tr069-identifier-semantics/design.md
- Lines: 1-96
- SHA256: 525042e83290763507c3e6d307b3230daa82cbad2db2d7e627a66cf037e4cdec

[TRUNCATED]

```md
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

### 3. CommandKey 由 Command ID 确定性派生

仅 Reboot、Download、Upload 在创建命令时生成 CommandKey。算法为：解析规范 UUID Command ID，将规范字符串中的四个连字符删除，得到 32 个小写十六进制字符。

```text
Command ID:  62a53a00-786f-4a45-b31f-b23f7d23bb6e
CommandKey:  62a53a00786f4a45b31fb23f7d23bb6e
```

重试创建新的 Command ID，因此自然得到新的 CommandKey，并通过 `retry_of` 关联原命令。历史 CommandKey 原样保留。完整 CommandKey 可以反推出 Command ID，这被接受为既定权衡；Command ID 只是弱化展示，不是秘密。

### 4. Trace ID 与 Session ID 完全分离

Trace ID 表示一次 HTTP 请求，只存在于请求上下文和结构化日志。GVA 与 core 内部统一使用 `TraceID/traceId`。外部仍接受常见的 `X-Request-ID`，入口将其映射为 Trace ID；缺失时生成 UUID。

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

```

Full source: openspec/changes/unify-tr069-identifier-semantics/design.md

## openspec/changes/unify-tr069-identifier-semantics/tasks.md

- Source: openspec/changes/unify-tr069-identifier-semantics/tasks.md
- Lines: 1-34
- SHA256: b0e7279547141aa69e115f77e9b1dff601d5cb165a35d7bb7b5cce821b4a1c31

```md
## 1. tr069-core 标识语义

- [ ] 1.1 为 `Request.TraceID` 和可观测 `TraceID` 传播编写失败测试，并用编译检查约束旧 Request ID 字段已删除
- [ ] 1.2 直接将 core 请求与可观测属性切换到 Trace ID 新字段，删除旧字段和兼容逻辑
- [ ] 1.3 为同一 CWMP Session 的多次 HTTP 请求编写 Trace ID 与 Session ID 分离测试并修正会话日志
- [ ] 1.4 将 `CommandContext.RequestID` 改为 `CWMPID`，将 `MarkSending` 参数语义改为 `cwmpID` 并通过 core 全量测试

## 2. GVA 数据模型与迁移

- [ ] 2.1 为停机执行的 `tr069_commands.request_id` 到 `cwmp_id` 列重命名及 XML 旧列删除编写数据库结构测试
- [ ] 2.2 将 GVA 命令模型、仓储、状态转换和查询关联切换到 `CWMPID/cwmp_id/cwmpId`
- [ ] 2.3 停止读写 XML Request ID，更新 XML sink、模型和 DTO，使其只返回报文实际 CWMP ID
- [ ] 2.4 更新 GVA 命令 API、服务和存储测试，确认新代码不依赖 `requestId`

## 3. CommandKey 派生与异步关联

- [ ] 3.1 为规范 UUID 派生 32 字符小写十六进制 CommandKey 编写失败测试
- [ ] 3.2 为 Reboot、Download、Upload 实现由 Command ID 确定性派生 CommandKey，普通 RPC 保持为空
- [ ] 3.3 验证重试生成新的 Command ID 与 CommandKey、`retry_of` 关联正确且历史 CommandKey 不被改写
- [ ] 3.4 验证 Reboot Inform 与 Download/Upload TransferComplete 继续通过 CommandKey 幂等关联原命令

## 4. GVA Trace 与用户界面

- [ ] 4.1 为 GVA HTTP Trace 中间件编写测试，将外部 `X-Request-ID` 映射为内部 `TraceID/traceId`
- [ ] 4.2 将 GVA Trace 上下文、调试路由和结构化日志字段改为 Trace ID，并确保 Trace ID 不进入命令/XML DTO
- [ ] 4.3 编写前端契约测试，要求列表、详情、XML 和成功提示均不渲染 Command ID 或 Request ID
- [ ] 4.4 更新 RPC 命令 UI：展示 CWMP ID，按状态显示未生成文案，仅对 Reboot、Download、Upload 展示 CommandKey

## 5. 验证与运行检查

- [ ] 5.1 运行 tr069-core 全量 Go 测试并确认旧 Request ID 字段和调用方式已完全移除
- [ ] 5.2 运行 GVA TR-069 插件全量 Go 测试、数据库结构切换测试和前端 contract tests
- [ ] 5.3 构建 GVA 前端与后端，确认 API JSON 中用户可见字段只使用 `cwmpId` 和 `traceId`
- [ ] 5.4 停止 GVA、执行数据库一次性结构切换，再重启前后端并验证命令详情、XML 详情及服务健康状态

```

## openspec/changes/unify-tr069-identifier-semantics/specs/tr069-identifier-semantics/spec.md

- Source: openspec/changes/unify-tr069-identifier-semantics/specs/tr069-identifier-semantics/spec.md
- Lines: 1-133
- SHA256: 1082a29f63008f6ac5eb2a2ce5f4cd57ce74697212d6d117111e2647aad85d44

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: Command ID 必须保持内部生命周期主键
系统 SHALL 使用 Command ID 作为 GVA 命令、事件、XML、队列状态和重试关系的内部稳定主键，但普通用户界面 SHALL NOT 展示、复制或在成功提示中输出 Command ID。

#### Scenario: 用户查看命令记录
- **WHEN** 用户打开 RPC 命令列表或命令详情
- **THEN** 页面不渲染 Command ID 的列、描述项或复制入口
- **AND** 后端仍能使用 Command ID 加载详情和执行受权限保护的重试操作

#### Scenario: 命令在发送前失败
- **WHEN** 命令在排队、等待设备或 core 构造阶段失败
- **THEN** 系统仍通过 Command ID 保存完整状态时间线和失败原因

### Requirement: CWMP ID 必须只表示 SOAP 请求响应关联
系统 SHALL 使用 `cwmp_id/CWMPID/cwmpId` 表示 SOAP `<cwmp:ID>`，并 SHALL NOT 使用 Request ID 命名该协议标识。

#### Scenario: 出站 RPC 成功构造并发送
- **WHEN** core 为 GVA 命令构造出站 CWMP RPC
- **THEN** core 将实际 SOAP CWMP ID 传给 `MarkSending`
- **AND** GVA 将其保存到命令的 `cwmp_id`
- **AND** 命令详情显示相同的 CWMP ID

#### Scenario: 命令尚未产生 SOAP 报文
- **WHEN** 命令仍在排队或等待设备
- **THEN** 命令的 CWMP ID 保持为空
- **AND** UI 显示“尚未生成”而不是伪造协议 ID

#### Scenario: core 构造命令失败
- **WHEN** core 在生成 SOAP 报文前构造失败
- **THEN** 命令详情显示“未生成（构造失败）”

### Requirement: XML 记录必须保存报文实际 CWMP ID
系统 SHALL 为每条已持久化的入站或出站 XML 保存报文实际 CWMP ID，并 SHALL NOT 在 XML DTO 或 UI 中提供 Request ID。

#### Scenario: 查看请求和响应 XML
- **WHEN** 用户打开命令的完整 XML 标签
- **THEN** 每条记录只展示方向、方法、时间、CWMP ID 和 XML payload
- **AND** 请求与响应展示各自报文中的真实 CWMP ID

#### Scenario: 查看设备主动异步报文
- **WHEN** TransferComplete 或 Inform 使用设备生成的 CWMP ID 到达
- **THEN** XML 记录保存该设备生成的 CWMP ID
- **AND** 系统不使用原命令请求的 CWMP ID 覆盖它

### Requirement: CommandKey 必须符合 32 字符派生规则
系统 SHALL 仅为 Reboot、Download、Upload 生成 CommandKey，并 SHALL 通过删除规范 UUID Command ID 的连字符得到 32 个小写十六进制字符。

#### Scenario: 创建异步关联命令
- **WHEN** GVA 创建 Reboot、Download 或 Upload 命令
- **THEN** CommandKey 等于 Command ID 删除连字符后的值
- **AND** CommandKey 长度为 32
- **AND** CommandKey 只包含小写十六进制字符

#### Scenario: 创建普通 RPC 命令
- **WHEN** GVA 创建不依赖异步完成通知的普通 RPC 命令
- **THEN** 系统不生成 CommandKey
- **AND** UI 不显示 CommandKey 描述项

#### Scenario: 重试异步关联命令
- **WHEN** 用户重试失败或超时的 Reboot、Download 或 Upload 命令
- **THEN** 系统创建新的 Command ID 和新的派生 CommandKey
- **AND** 新命令通过 `retry_of` 关联原命令

#### Scenario: 读取历史命令
- **WHEN** 系统读取使用旧规则生成的历史 CommandKey
- **THEN** 系统保持其原值并继续支持已有异步关联

### Requirement: Trace ID 必须只追踪单次 HTTP 请求
GVA 与 core SHALL 使用 `TraceID/traceId` 表示单次 HTTP 请求链路，Trace ID SHALL 只存在于请求上下文和日志，不得写入命令详情或 XML 记录。

#### Scenario: 请求携带外部链路头
- **WHEN** HTTP 请求携带 `X-Request-ID`
- **THEN** 入口将该值映射为内部 Trace ID
- **AND** core 可观测事件和 GVA 结构化日志使用 `traceId` 字段

#### Scenario: 请求未携带链路头
- **WHEN** HTTP 请求没有 `X-Request-ID`
- **THEN** 入口生成新的 UUID Trace ID
- **AND** 该值只覆盖当前 HTTP 请求链路

```

Full source: openspec/changes/unify-tr069-identifier-semantics/specs/tr069-identifier-semantics/spec.md
