# Comet Design Handoff

- Change: enable-tr069-pm-mr-ingress
- Phase: design
- Mode: compact
- Context hash: ca40fe619d62db8febb67a45c370fd1478bcb5b069d265bc859daaca00ed1532

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/enable-tr069-pm-mr-ingress/proposal.md

- Source: openspec/changes/enable-tr069-pm-mr-ingress/proposal.md
- Lines: 1-31
- SHA256: 6867add8ec1e822cddb811dc62c8015546d81d2efb11b768221e2b2c987a7ac5

```md
## Why

现有 TR-069 文件通道只启用了固定地址 `/acs/log`，PM 和 MR 虽然已有配置骨架，但没有独立认证、接收、任务和管理能力。基站已经提供本地 PM/MR URL、用户名、密码和周期参数，GVA 应在不修改这些设备参数的前提下接收并校验定时上传。

## What Changes

- 在现有 7458 文件入口启用固定路径 `PUT/POST /acs/pm[/*filename]` 和 `PUT/POST /acs/mr[/*filename]`。
- PM、MR分别使用全局 PM、MR凭据，支持用户名和密码同时为空的无认证模式以及 Basic/Digest 自动兼容。
- 复用最近成功 Inform 的来源 IP绑定解析设备；没有唯一已注册设备时返回 `403`。
- PM、MR上传只产生 `PERIODIC` 传输任务，不提供主动 Upload RPC或“立即上传”操作。
- 复用 LOG 的流式限长、并发控制、对象存储、SHA-256、一致性恢复、保留期和安全审计能力。
- 增加 PM/MR 文件查询和权限控制，不解析文件内部业务内容。
- GVA 不读取、设置或下发 `Device.FAP.PerfMgmt.*` 与 `Device.FAP.MRMgmt.*` 的 URL、用户名、密码或周期参数。

## Capabilities

### New Capabilities

- `tr069-pm-mr-file-ingress`: 定义 PM/MR 固定文件入口、独立凭据、设备解析、周期任务、制品管理和权限行为。

### Modified Capabilities

- 无。

## Impact

- 后端：文件通道注册、认证 Provider、传输任务分类、制品查询 API、权限初始化和配置校验。
- 前端：PM、MR 文件管理入口、列表、筛选和下载操作。
- 数据库与对象存储：复用现有传输表，以 channel 区分 LOG/PM/MR，并使用独立存储前缀和保留策略。
- 基站：用户继续人工配置固定地址 `/acs/pm`、`/acs/mr` 及本地凭据；无需 URL 迁移。
- 测试：固定路径隔离、跨模块凭据拒绝、IP 歧义拒绝、流式上传和两台 BS Docker 周期上传验收。

```

## openspec/changes/enable-tr069-pm-mr-ingress/design.md

- Source: openspec/changes/enable-tr069-pm-mr-ingress/design.md
- Lines: 1-64
- SHA256: e481c48f6fb62794d4ddbaf63200203f2ad160f3db71c9d011440ccbb214cdfd

```md
## Context

现有 `add-tr069-log-collection` change 已建立7458同端口文件通道、Basic/Digest、Inform/IP唯一设备解析、ACTIVE/PERIODIC任务、流式对象存储和日志管理页面。配置中已经预留 `/acs/pm` 与 `/acs/mr`，但默认禁用。两台基站当前 `Device.FAP.PerfMgmt.Config.1.URL` 和 `Device.FAP.MRMgmt.Config.1.MrUrl` 均为空，后续由用户人工设置固定入口与本地凭据。

## Goals / Non-Goals

**Goals:**

- 启用固定PM、MR入口并分别使用全局PM、MR凭据。
- 复用LOG文件接收、设备解析、对象存储和任务状态机。
- 只把成功认证且唯一映射到已注册设备的文件记录为PERIODIC。
- 提供按channel隔离的查询、下载、权限、限制、保留期和审计。

**Non-Goals:**

- 不修改基站PM/MR URL、用户名、密码、周期或启用参数。
- 不实现主动PM/MR Upload RPC或页面操作。
- 不修改固定地址，不在URL加入设备标识。
- 不解析PM/MR文件内部格式或建立指标/测量业务模型。
- 不解决多设备共享同一可见源IP的归属问题。

## Decisions

### 1. 复用通道注册与固定路径

在现有FileIngress channel registry中启用PM和MR，路径固定为`/acs/pm`与`/acs/mr`，并兼容尾斜杠、raw PUT/POST、PUT单段文件名和现有multipart流式策略。文件路由继续绕过CWMP RawDump与XML解析。

相比建立独立服务，通道方式复用已验证的7458监听、限制、存储和恢复逻辑；channel policy只提供凭据映射、路径、大小、并发、超时、保留期和存储前缀。

### 2. 独立凭据Provider与统一认证器

PM_UPLOAD与MR_UPLOAD通道分别解析PM、MR Profile。空Profile关闭认证但不关闭业务；非空Profile支持Basic/Digest。认证成功只授予当前channel，不允许PM凭据访问MR，也不从共享用户名推导设备。

### 3. Inform/IP唯一设备归属

文件认证成功后沿用最近成功Inform的Redis绑定和MySQL回退。只有来源IP在窗口内唯一对应一个已注册设备时才接收；无候选、过期或歧义统一返回403。固定URL和全局凭据无法区分同一NAT后的多设备，因此拒绝而不保存未知文件。

### 4. PM/MR始终为PERIODIC

PM、MR入口不查询或创建ACTIVE Upload命令。文件通过认证、设备解析和准入后，分别创建channel=PM/MR、source=PERIODIC任务，并使用文件成功保存作为完成事实。若设备另发AutonomousTransferComplete，只追加可关联事件，不改变认证事实。

### 5. 共享制品模型、按channel隔离管理

继续使用`tr069_transfer_tasks`、`tr069_artifacts`和`tr069_transfer_events`，以channel索引区分类型；对象键使用pm/mr独立前缀。管理API必须显式过滤允许channel并重复校验JWT、Casbin、设备数据权限与AVAILABLE状态。前端可共用文件列表组件，但菜单、标题、权限和下载操作独立。

## Risks / Trade-offs

- [PM/MR周期较长导致IP绑定过期] → 每次成功Inform刷新绑定，并允许在配置窗口内从MySQL回填；过期时安全拒绝。
- [不同通道错误复用凭据] → Provider按channel解析，认证成功产生channel-scoped principal并有交叉拒绝测试。
- [固定URL在NAT后无法归属] → 明确要求唯一管理IP；歧义返回403，不创建未知文件。
- [大文件或并发影响CWMP] → 沿用流式限长、通道/设备并发、超时和对象存储Abort机制。

## Migration Plan

1. 先部署并验证`secure-tr069-credential-authentication`与修订后的LOG通道。
2. 数据库无需新增制品表，仅增加PM/MR菜单、权限和必要channel索引/种子数据。
3. 在GVA设置PM、MR凭据并保持通道可用；凭据为空时先以无认证模式验路。
4. 用户在第一台BS设置`/acs/pm`、`/acs/mr`和本地凭据，验证周期文件与归属。
5. 第二台BS重复验证并确认不同IP不会串设备；再验证错误/交叉凭据、超限与超时。
6. 回滚时禁用PM/MR channel并保留表、对象与审计；LOG和CWMP不受影响。

## Open Questions

无。PM/MR文件内容保持不透明；真实基站配置和周期触发属于部署验收。

```

## openspec/changes/enable-tr069-pm-mr-ingress/tasks.md

- Source: openspec/changes/enable-tr069-pm-mr-ingress/tasks.md
- Lines: 1-33
- SHA256: 6d168a75f10d859bbe9f5f54e54c727d34bd32e5780a554c685f85341f2890b6

```md
## 1. 通道配置与认证

- [ ] 1.1 为PM/MR固定路径、独立Profile、空凭据和跨模块拒绝编写失败测试
- [ ] 1.2 启用PM、MR channel配置并分别映射PM_UPLOAD、MR_UPLOAD凭据Provider
- [ ] 1.3 注册`/acs/pm`、`/acs/mr`及兼容尾斜杠/单文件名PUT/POST路由
- [ ] 1.4 验证PM/MR文件路由绕过CWMP RawDump和XML解析

## 2. 设备解析与周期任务

- [ ] 2.1 添加唯一IP绑定、无绑定、过期和NAT歧义的PM/MR接收测试
- [ ] 2.2 复用Inform/Redis/MySQL设备解析器并保持歧义403行为
- [ ] 2.3 为PM/MR创建channel隔离的PERIODIC任务和AVAILABLE制品
- [ ] 2.4 添加测试证明PM/MR没有主动Upload API、命令或WAITING_FILE任务

## 3. 流式存储与可靠性

- [ ] 3.1 复用限长、并发、超时、SHA-256、幂等和Abort流程并补充PM/MR参数化测试
- [ ] 3.2 为PM、MR配置独立对象前缀、文件上限、保留期和并发限制
- [ ] 3.3 扩展协调器与清理器测试，证明channel不会串数据或删除错误对象

## 4. 管理API与前端

- [ ] 4.1 实现PM/MR channel过滤的列表、下载和权限校验API
- [ ] 4.2 添加数据范围、AVAILABLE状态、跨channel与秘密字段省略测试
- [ ] 4.3 复用文件页面组件增加PM、MR菜单、筛选和下载，不显示立即上传操作
- [ ] 4.4 添加前端路由、权限、分页、下载和主题测试

## 5. 文档与真实基站验收

- [ ] 5.1 文档化固定`/acs/pm`、`/acs/mr`、基站人工参数配置、唯一IP和HTTPS要求
- [ ] 5.2 运行文件入口、对象存储、TR-069后端和前端回归
- [ ] 5.3 在第一台BS配置PM/MR地址与凭据并验证周期上传、任务和文件归属
- [ ] 5.4 在第二台BS重复验证并测试错误凭据、跨模块凭据、无绑定和歧义拒绝

```

## openspec/changes/enable-tr069-pm-mr-ingress/specs/tr069-pm-mr-file-ingress/spec.md

- Source: openspec/changes/enable-tr069-pm-mr-ingress/specs/tr069-pm-mr-file-ingress/spec.md
- Lines: 1-86
- SHA256: ee5b0cef32ad0e25a84becb7037fe6685b2406fab9802ceb64a9d902d1ee3a8a

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: PM与MR固定文件入口
系统 SHALL 在TR-069的7458监听器上提供`PUT/POST /acs/pm[/*filename]`和`PUT/POST /acs/mr[/*filename]`，且文件请求不得进入CWMP XML解析或RawDump。

#### Scenario: PM raw PUT
- **WHEN** 基站向`/acs/pm`或单段文件名后缀发送合法PUT文件流
- **THEN** 系统通过PM通道流式处理并返回成功状态

#### Scenario: MR multipart POST
- **WHEN** 基站向`/acs/mr`发送兼容的multipart `file` POST
- **THEN** 系统通过MR通道流式读取文件且不创建整包内存或临时文件

#### Scenario: 非法方法或路径
- **WHEN** 请求使用不支持的方法、多段文件后缀或跨通道路由
- **THEN** 系统返回405或404且不创建任务或对象

### Requirement: PM与MR独立凭据
PM入口 MUST 只使用PM Profile，MR入口 MUST 只使用MR Profile，并自动兼容Basic/Digest或对应空凭据无认证模式。

#### Scenario: 正确PM凭据
- **WHEN** 请求使用当前PM凭据访问`/acs/pm`
- **THEN** 认证授予PM_UPLOAD principal并继续设备解析

#### Scenario: 跨模块凭据
- **WHEN** 请求使用LOG或MR凭据访问`/acs/pm`
- **THEN** 系统返回401且不读取文件正文、不创建对象

#### Scenario: PM认证关闭
- **WHEN** PM用户名和密码都为空
- **THEN** `/acs/pm`保持业务可用且不要求Authorization

### Requirement: 唯一已注册设备解析
系统 MUST 在认证成功后根据最近成功Inform的来源IP绑定解析唯一已注册设备。

#### Scenario: 唯一匹配
- **WHEN** 上传来源IP在有效窗口内唯一映射到一个已注册设备
- **THEN** 文件归属该设备并继续接收

#### Scenario: 无匹配或绑定过期
- **WHEN** 来源IP没有有效设备绑定
- **THEN** 系统返回403且不创建可用制品

#### Scenario: NAT歧义
- **WHEN** 来源IP映射到多个候选设备
- **THEN** 系统返回403并记录DEVICE_AMBIGUOUS事件，不猜测归属

### Requirement: PM与MR仅周期任务
系统 SHALL 把成功接收的PM和MR文件分别记录为channel=PM/MR、source=PERIODIC，不得为两类通道创建主动Upload命令。

#### Scenario: PM周期上传
- **WHEN** PM文件成功保存并校验
- **THEN** 系统创建或完成PERIODIC PM任务及AVAILABLE制品

#### Scenario: 禁止主动上传
- **WHEN** 管理用户查看PM或MR文件页面/API
- **THEN** 系统不提供立即上传操作或对应Upload命令端点

### Requirement: 流式资源保护与一致性
PM/MR SHALL 复用文件通道的限长、并发、超时、SHA-256、幂等、Abort、协调和保留期行为。

#### Scenario: 文件超限
- **WHEN** 声明长度或流式读取超过通道上限
- **THEN** 系统返回413、终止对象上传并且不产生AVAILABLE制品

#### Scenario: 客户端中断
- **WHEN** 上传在完成前断开或超时
- **THEN** 系统释放并发资源、Abort存储写入并记录FAILED任务

### Requirement: PM与MR管理权限隔离
系统 SHALL 提供按channel过滤的PM/MR列表和下载，并再次校验用户权限、设备数据范围和制品AVAILABLE状态。

#### Scenario: PM列表
- **WHEN** 有权限用户查询PM文件
- **THEN** 只返回其设备数据范围内的AVAILABLE PM制品，不返回MR/LOG或存储秘密

#### Scenario: 越权下载
- **WHEN** 用户尝试下载无数据权限设备或不同channel的文件
- **THEN** 系统拒绝请求且不返回对象内容或对象键


```

Full source: openspec/changes/enable-tr069-pm-mr-ingress/specs/tr069-pm-mr-file-ingress/spec.md
