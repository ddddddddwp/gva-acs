# Comet Design Handoff

- Change: secure-tr069-credential-authentication
- Phase: design
- Mode: compact
- Context hash: 5561f4735646c9e99bd41a0f24945c270385694bcb317a3a8709c01ae8bf6395

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/secure-tr069-credential-authentication/proposal.md

- Source: openspec/changes/secure-tr069-credential-authentication/proposal.md
- Lines: 1-33
- SHA256: 128c29a1b7322ae9c4222235939d67668cfa9517609a50e744e78869c6910cf3

```md
## Why

当前 GVA 会为设备自动生成并通过 TR-069 下发 Connection Request 用户名和密码，文件上传也会把共享凭据写入 Upload RPC。这覆盖了基站侧由用户维护凭据的职责，并扩大了密码进入命令、XML 和日志链路的风险。系统需要改为“用户分别配置、GVA 只在真实协议交互中校验”的认证模型。

## What Changes

- 新增 Connection、LOG、PM、MR 四套全局凭据配置，用户名和密码必须同时为空或同时非空。
- 新增可复用的 Basic/Digest 认证核心、加密凭据存储、版本化验证状态和不含秘密的审计记录。
- Connection 凭据默认同时供设备到 ACS 的 CWMP 接入认证和 ACS 到设备的 Connection Request 使用，但两个通道独立记录验证结果。
- 在 CWMP Handler 前增加设备接入认证；认证失败的 Inform 不解析、不注册、不更新设备。
- 移除自动生成、自动调度和自动下发 Connection Request 凭据的行为；历史自动凭据保留为只读审计数据。
- Connection Request 只在实际请求时使用用户配置的共享凭据，并自动兼容 Basic/Digest challenge。
- 新增全局凭据管理页面和设备认证状态展示；密码不回显，也不提供脱离真实协议动作的手动校验。
- **BREAKING**：`autoProvisionCredentials` 不再启用凭据下发；旧 YAML 文件上传凭据不再作为认证事实来源。

## Capabilities

### New Capabilities

- `tr069-credential-management`: 定义四套全局凭据、通道映射、原子保存、加密存储、版本轮换、状态和审计契约。
- `tr069-cwmp-access-authentication`: 定义 CWMP 设备接入认证、失败阻断、Connection Request 共享凭据和旧自动配置停用行为。

### Modified Capabilities

- 无。

## Impact

- 后端：TR-069 配置、模型、迁移、仓储、凭据服务、Basic/Digest 校验、CWMP 路由中间件、Connection Request 客户端、Inform 注册链路和安全日志。
- 前端：全局凭据管理页、设备详情认证状态、密码更新与清除交互。
- 数据库：新增凭据配置、认证通道映射、设备通道状态和认证审计表；旧连接配置表保留但停止参与运行时解析。
- 部署：需要独立主加密密钥；Basic 在生产环境必须由 HTTPS 保护。
- 测试：认证协议向量、未认证注册阻断、凭据轮换、旧数据迁移、脱敏和两台 BS Docker 的真实交互。

```

## openspec/changes/secure-tr069-credential-authentication/design.md

- Source: openspec/changes/secure-tr069-credential-authentication/design.md
- Lines: 1-85
- SHA256: ac31efd6dee954ee08a03c87d56222d0d7d6bd6ab14e6c6d334abe014b758118

[TRUNCATED]

```md
## Context

TR-069 插件当前从 Inform/GPV 收集 `ConnectionRequestUsername/Password`，在凭据缺失时生成随机值并提交 SYSTEM `SetParameterValues`；LOG Upload API 又把配置文件中的共享用户名和密码写入 Upload RPC。设备密码属于写敏感参数，读取通常只能得到空值，GVA 无法通过读取比较可靠验证。确认后的产品边界是：用户在 GVA 和基站分别配置同一套全局凭据，GVA 只在真实 HTTP/CWMP 交互中验证，绝不修改设备凭据。

四个用户可见凭据 Profile 为 CONNECTION、LOG、PM、MR；五个内部认证通道为 CWMP_ACCESS、CONNECTION_REQUEST、LOG_UPLOAD、PM_UPLOAD、MR_UPLOAD。前两个通道当前共同引用 CONNECTION Profile，但映射关系独立，以便以后拆分而不重写认证器。

## Goals / Non-Goals

**Goals:**

- 提供四套全局、可动态更新、加密存储的用户名和密码。
- 在 TR-069 插件内部建立共享认证核心，并由各协议入口使用薄适配层。
- 在任何 Inform 解析和设备 Upsert 之前阻断未认证 CWMP 请求。
- Connection Request 使用用户配置的共享凭据，删除自动配置前置流程。
- 通过真实协议动作维护每设备、每通道、每凭据版本的验证状态。
- 对 API、日志、Trace、命令 JSON/XML 和审计进行统一秘密脱敏。

**Non-Goals:**

- 不实现每设备独立凭据。
- 本阶段不向用户提供 Connection 双向凭据拆分开关。
- 不读取或比较设备密码参数，不提供手动“测试密码”操作。
- 不删除既有设备、历史命令或历史自动生成凭据。
- 不在凭据页面暴露 Basic/Digest 选择；协议自动兼容。

## Decisions

### 1. 凭据 Profile 与认证通道分离

新增全局 `CredentialProfile`，以稳定 key 标识 CONNECTION、LOG、PM、MR；新增 `AuthChannelBinding`，把五个认证通道映射到 Profile。初始数据让 CWMP_ACCESS 和 CONNECTION_REQUEST 都引用 CONNECTION。相比把两个方向硬编码到同一配置结构，映射模型允许未来新增独立 Profile 后只更新绑定和 UI。

凭据保存只接受两种原子状态：用户名和密码都为空，或两者都非空。API 使用明确的 set/clear 语义；密码字段缺失表示请求非法而不是“保留旧值”，修改用户名必须重新提交密码。密码加密后存入数据库，读取 API 只返回 `passwordConfigured`。

### 2. 版本化状态与可信审计分离

每次 set/clear 都递增 Profile revision。`DeviceAuthState` 以 device_id + channel 唯一，记录成功使用的 revision、最后成功时间和非敏感错误。Profile 变更后引用它的状态进入待验证；认证关闭时状态为未启用。

失败请求在认证前没有可信设备身份，只记录 channel、来源 IP、结果、时间和可选的未可信声明，不修改正式设备状态。已有设备会因 `LastInform` 不再更新而离线。只有成功认证且成功处理 Inform、Connection Request 或文件传输后才更新正式状态。

### 3. 共享认证核心、通道薄适配

认证核心接受 Credential Provider、允许方案和请求元数据，支持 Basic 常量时间比较以及 Digest MD5/MD5-sess、qop=auth、nonce TTL 和 Redis 防重放。设备接入与文件入口分别拥有薄中间件，Connection Request 客户端复用 challenge 解析和凭据 Provider，但不把路由、设备解析或传输任务塞入通用认证器。

用户名和密码都为空时 Provider 返回认证关闭，中间件直接放行。配置非空时，无 Authorization 的首次请求返回 401 challenge，不计为错误；携带无效 Authorization 后仍失败才写拒绝审计。开发环境兼容 HTTP，生产环境启用 Basic 时必须通过 HTTPS。

### 4. CWMP 认证在解析和注册之前

7458 的 `/acs` 和兼容根路由在 RawDump/CWMP Handler 前挂载请求大小限制与 `CWMPAccessAuthMiddleware`。失败响应不读取 SOAP body、不调用引擎、不执行 `UpsertFromInform`。认证成功后把 channel、profile revision 和来源 IP写入 context；只有 Inform 成功持久化后才将设备 CWMP_ACCESS 状态设为成功。

这种顺序避免信任未经认证的 DeviceIdStruct。失败请求只记录 IP；不会根据请求体序列号把其他设备标成失败。

### 5. Connection Request 删除自动 Provisioning

移除 `ConnectionCredentialProvisioner` 的调度入口，`autoProvisionCredentials` 保留为废弃兼容字段并输出无秘密警告，但不再触发命令。`ConnectionProfileRepository` 继续收集设备上报的 ConnectionRequestURL，不再把上报用户名/密码作为运行时认证事实，也不再要求 per-device 密文。

Connection Request 客户端从 CONNECTION_REQUEST 通道解析 Profile：凭据关闭时发无认证请求；启用时根据设备 401 challenge 使用 Basic 或 Digest。HTTP认证成功与随后收到 `6 CONNECTION REQUEST` Inform 分别记录，便于区分认证、网络和设备回呼问题。

### 6. API 与页面

新增全局凭据 list/set/clear API，返回四个 Profile 的启用状态、revision、修改时间和汇总验证数，不返回密码。页面以四张卡展示 Connection、LOG、PM、MR；Connection 明示被两个方向共享。设备详情展示五个通道的未启用、待验证、成功状态和最近安全事件，不提供人工校验按钮。

### 7. 全链路秘密保护

密码密文使用部署主密钥和 key version；缺少主密钥时仍允许全空 Profile，但拒绝保存非空凭据。所有 Authorization 头、SOAP 中任意 Username/Password 参数、命令 JSON/XML和错误详情使用统一 redact 规则。Upload RPC 在 LOG change 中改为空凭据，因此持久化命令不需要秘密 hydrate。

## Risks / Trade-offs

- [全局凭据泄漏影响所有设备] → 强制加密、最小回显、HTTPS、版本轮换和审计；未来可通过通道映射扩展每设备 Profile。
- [Basic 在 HTTP 上可恢复凭据] → 仅为现有 BS 开发兼容保留；生产部署要求 HTTPS。
- [认证失败无法可信关联设备] → 只记来源 IP，设备以最后成功 Inform 判断在线，不解析未认证 SOAP 身份。
- [旧设备仍保留随机 Connection Request 密码] → 新全局凭据启用前由用户在设备侧人工更新；GVA 不尝试自动迁移不同的旧值。
- [凭据轮换造成短暂不可用] → revision 使状态明确回到待验证，旧设备数据不删除，成功交互后自动恢复。

## Migration Plan

1. 新增凭据、绑定、状态和审计表，种子化四个 Profile 与五个通道映射。
2. 部署 API/UI 和主密钥校验；初始 Profile 全空，保持无认证兼容。
3. 停止自动 Provisioner，给旧 Connection Profile 标记 LEGACY/INACTIVE，但保留表与命令历史。
4. 用户先在所有基站人工设置共享 Connection 凭据，再在 GVA 原子保存同一凭据。
5. 启用 CWMP中间件并观察两台 Docker BS 的认证 Inform；失败不改变历史设备数据。

```

Full source: openspec/changes/secure-tr069-credential-authentication/design.md

## openspec/changes/secure-tr069-credential-authentication/tasks.md

- Source: openspec/changes/secure-tr069-credential-authentication/tasks.md
- Lines: 1-44
- SHA256: 6f0708bf6aed74d58331d45759e23d5c7fe4256e08e270bf52ac35f8b7d83ca8

```md
## 1. 凭据数据基础

- [ ] 1.1 为四个全局Profile、五个认证通道映射、设备通道状态和审计模型编写失败测试
- [ ] 1.2 实现模型、唯一索引、种子数据和TR-069插件AutoMigrate注册
- [ ] 1.3 实现凭据加密仓储、主密钥校验、set/clear原子语义和revision递增
- [ ] 1.4 实现通道到Profile解析、状态失效和不含密码的DTO/API服务

## 2. 共享HTTP认证核心

- [ ] 2.1 为Basic、Digest MD5/MD5-sess、qop=auth、nonce过期/重放、空Profile编写协议向量测试
- [ ] 2.2 抽取通道无关CredentialProvider、Authenticator和认证Principal
- [ ] 2.3 实现Basic常量时间比较与Digest challenge/校验，复用Redis nonce store
- [ ] 2.4 实现失败审计、正常challenge区分、Authorization和错误脱敏

## 3. CWMP设备接入认证

- [ ] 3.1 添加失败测试，证明401请求不会进入CWMP Handler、解析Inform或创建设备
- [ ] 3.2 在7458 CWMP路由前挂载接入认证中间件并传递安全context
- [ ] 3.3 在Inform成功持久化后更新CWMP_ACCESS状态，失败只记录来源IP审计
- [ ] 3.4 验证空Connection Profile保持现有无认证CWMP行为

## 4. Connection Request改造

- [ ] 4.1 添加测试，证明缺失设备凭据不再生成`gva-connection-request` SPV命令
- [ ] 4.2 删除/禁用自动Provisioner调度入口并为废弃配置输出安全警告
- [ ] 4.3 让Connection Request客户端通过通道Provider读取全局Connection凭据并兼容Basic/Digest
- [ ] 4.4 分别记录HTTP认证和后续Connection Request Inform结果
- [ ] 4.5 将旧AUTO Profile标记LEGACY/INACTIVE并验证运行时不再读取旧密文

## 5. 管理API与前端

- [ ] 5.1 实现四Profile查询、原子设置、显式清除API及JWT/Casbin权限
- [ ] 5.2 为半配置拒绝、密码不回显、权限和revision状态失效添加API测试
- [ ] 5.3 实现四卡凭据管理页面，包含Connection双通道共享说明和显式清除确认
- [ ] 5.4 在设备详情展示五通道状态、最近成功与不可信安全事件，不增加手动校验按钮
- [ ] 5.5 添加前端API、表单校验、密码处理和状态展示测试

## 6. 安全、迁移与验收

- [ ] 6.1 扩展XML、命令、Trace、RawDump和操作日志脱敏测试，覆盖通用Username/Password与Authorization
- [ ] 6.2 更新配置、部署文档和数据库迁移说明，明确主密钥、HTTPS和全局凭据风险
- [ ] 6.3 运行TR-069后端、前端、race/static回归并确认无明文秘密
- [ ] 6.4 使用两台Docker BS验证正确/错误/空Connection接入与已有设备保留行为
- [ ] 6.5 验证实际Connection Request使用共享凭据且数据库不产生新的凭据下发命令

```

## openspec/changes/secure-tr069-credential-authentication/specs/tr069-credential-management/spec.md

- Source: openspec/changes/secure-tr069-credential-authentication/specs/tr069-credential-management/spec.md
- Lines: 1-90
- SHA256: 4e4be57bb1be000c4be73b31be36f3ab8fe556ce6dab38ddbea5c719b0310cc2

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: 全局凭据 Profile
系统 SHALL 提供 CONNECTION、LOG、PM、MR 四个全局凭据 Profile，并通过独立认证通道映射让 CWMP_ACCESS 与 CONNECTION_REQUEST 默认引用同一个 CONNECTION Profile。

#### Scenario: 默认通道映射
- **WHEN** 系统初始化凭据数据
- **THEN** 五个认证通道存在，且两个 Connection 方向共同引用 CONNECTION，三个文件通道分别引用 LOG、PM、MR

#### Scenario: 未来拆分兼容
- **WHEN** 后续为某认证通道配置不同 Profile
- **THEN** 认证调用方通过通道解析凭据，不需要改变中间件或 Connection Request 客户端接口

### Requirement: 用户名密码原子配置
系统 MUST 只接受用户名和密码同时为空或同时非空的配置，不得持久化半配置状态。

#### Scenario: 关闭认证
- **WHEN** 管理员明确清除某 Profile 的用户名和密码
- **THEN** 系统原子清除两者并将引用通道标记为未启用

#### Scenario: 启用认证
- **WHEN** 管理员同时提交非空用户名和密码
- **THEN** 系统原子保存新版本并将引用通道标记为待验证

#### Scenario: 拒绝半配置
- **WHEN** 请求只提供用户名或只提供密码
- **THEN** 系统拒绝请求且保持原 Profile 不变

### Requirement: 密码加密与最小回显
系统 MUST 使用部署主密钥加密非空密码，且所有查询 API SHALL 仅返回是否已配置密码，不得返回明文或密文。

#### Scenario: 主密钥缺失
- **WHEN** 主密钥不可用且管理员尝试保存非空密码
- **THEN** 系统拒绝保存并返回不含秘密的配置错误

#### Scenario: 查询凭据
- **WHEN** 管理员读取凭据 Profile
- **THEN** 响应包含 username、passwordConfigured、revision和时间，但不包含任何密码表示

### Requirement: 凭据版本与设备验证状态
系统 SHALL 为每次凭据 set/clear 增加 revision，并按设备和认证通道维护与当前 revision 对应的正式验证状态。

#### Scenario: Connection 轮换
- **WHEN** CONNECTION Profile revision改变
- **THEN** CWMP_ACCESS 与 CONNECTION_REQUEST 的设备状态都变为待验证，LOG、PM、MR状态不变

#### Scenario: 单文件模块轮换
- **WHEN** PM Profile revision改变
- **THEN** 只有 PM_UPLOAD 的设备状态变为待验证

#### Scenario: 真实动作验证成功
- **WHEN** 某设备使用当前 revision 完成对应协议动作
- **THEN** 系统记录该设备该通道验证成功及最后成功时间

### Requirement: Basic与Digest自动兼容
共享认证核心 SHALL 支持 HTTP Basic 以及 Digest MD5、MD5-sess、qop=auth，并校验 nonce有效期和重放。

#### Scenario: 首次挑战
- **WHEN** 启用认证的入口收到没有 Authorization 的请求
- **THEN** 系统返回401和兼容挑战，且不把正常挑战记为密码错误

#### Scenario: 无效认证
- **WHEN** 请求携带错误 Basic或Digest认证
- **THEN** 系统以常量时间校验失败、返回401并记录不含秘密的拒绝审计

#### Scenario: 认证关闭
- **WHEN** Profile用户名和密码都为空
- **THEN** 对应入口不要求Authorization并继续业务处理

### Requirement: 认证审计不信任失败身份
系统 MUST 将认证审计与正式设备状态分离，认证失败前不得信任请求声明的设备身份。

#### Scenario: 未认证CWMP失败
- **WHEN** CWMP请求认证失败
- **THEN** 审计只记录通道、来源IP、结果和时间，不修改任何设备正式状态

#### Scenario: 秘密脱敏
- **WHEN** 系统记录请求、命令、Trace、XML或错误
- **THEN** Authorization和所有密码值均被删除或替换为固定占位符


```

Full source: openspec/changes/secure-tr069-credential-authentication/specs/tr069-credential-management/spec.md

## openspec/changes/secure-tr069-credential-authentication/specs/tr069-cwmp-access-authentication/spec.md

- Source: openspec/changes/secure-tr069-credential-authentication/specs/tr069-cwmp-access-authentication/spec.md
- Lines: 1-63
- SHA256: 2f0792094cf1ae37c5ab5d6cf64fb7e37fd27096a710626816745a6f0d7b7f92

```md
## ADDED Requirements

### Requirement: CWMP注册前认证
系统 MUST 在读取和解析CWMP SOAP正文、调用TR-069引擎或注册设备之前完成设备接入HTTP认证。

#### Scenario: 新设备凭据错误
- **WHEN** 未注册设备向ACS发送认证失败的Inform请求
- **THEN** 系统返回401，且不创建设备、参数值、IP绑定或命令

#### Scenario: 已有设备凭据错误
- **WHEN** 已注册设备发送认证失败的Inform请求
- **THEN** 系统不更新LastInform、IP、版本或参数，并保留既有历史数据

#### Scenario: 认证成功
- **WHEN** 请求通过当前CONNECTION凭据认证且Inform处理成功
- **THEN** 系统注册或更新设备，并记录CWMP_ACCESS当前revision验证成功

### Requirement: CWMP无认证模式
当CONNECTION用户名和密码同时为空时，系统 SHALL 保持CWMP入口可用且不要求Authorization。

#### Scenario: 空凭据Inform
- **WHEN** CONNECTION Profile关闭且设备发送合法Inform
- **THEN** 系统按现有无认证行为处理并把通道显示为未启用

### Requirement: Connection Request共享用户凭据
系统 SHALL 让Connection Request客户端通过CONNECTION_REQUEST通道读取全局CONNECTION凭据，并在真实请求中响应设备Basic或Digest challenge。

#### Scenario: 有认证Connection Request
- **WHEN** 设备ConnectionRequestURL返回认证challenge且全局Connection凭据非空
- **THEN** GVA使用共享凭据完成认证，并分别记录HTTP认证结果与后续Connection Request Inform结果

#### Scenario: 无认证Connection Request
- **WHEN** Connection凭据为空
- **THEN** GVA发送不带Authorization的请求，设备若要求认证则记录失败而不生成凭据

### Requirement: 禁止自动Provisioning
系统 MUST NOT 生成、调度、下发或重试设置设备ConnectionRequestUsername/Password的命令。

#### Scenario: Inform缺少Connection Request凭据
- **WHEN** 设备Inform包含ConnectionRequestURL但不包含用户名或密码
- **THEN** 系统只保存URL，不创建`gva-connection-request` SetParameterValues命令

#### Scenario: 废弃配置仍为true
- **WHEN** 旧配置`autoProvisionCredentials=true`
- **THEN** 系统输出不含秘密的废弃警告并忽略该开关

### Requirement: 历史自动凭据只读保留
系统 SHALL 保留旧每设备Connection Profile和历史命令用于审计，但不得在新Connection Request运行时使用旧自动密文。

#### Scenario: 存在READY AUTO Profile
- **WHEN** 新版本处理已有AUTO连接Profile的设备
- **THEN** 运行时只使用全局CONNECTION Profile，并把旧记录视为LEGACY/INACTIVE

### Requirement: 认证数据全链路脱敏
系统 MUST 防止Connection密码和Authorization进入数据库命令载荷、XML详情、普通日志和Trace。

#### Scenario: RawDump开启
- **WHEN** 开启CWMP RawDump或InfoLog并发生认证交互
- **THEN** 输出中不包含Basic token、Digest response、用户名关联密码值或任何密码明文

#### Scenario: 命令历史查询
- **WHEN** 管理员查询历史或新Connection命令
- **THEN** API不得返回可用于认证的密码、密文或Authorization

```
