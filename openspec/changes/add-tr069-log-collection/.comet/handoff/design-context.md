# Comet Design Handoff

- Change: add-tr069-log-collection
- Phase: design
- Mode: compact
- Context hash: fec5d872d98d6d425133492c489d2eea59cbce58990407b9337d2f595d00f1d5

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/add-tr069-log-collection/proposal.md

- Source: openspec/changes/add-tr069-log-collection/proposal.md
- Lines: 1-35
- SHA256: 04857752aebf615e164494f72d6252845c4af063351802f015fb895aa4bdaa40

```md
## Why

BS 基站会通过 TR-069 文件传输把压缩日志上传到 ACS，但当前 GVA/TR-069 插件只有 CWMP XML 会话和 RPC 命令记录，没有安全、低内存的文件接收通道，也没有可按设备追踪和下载日志的管理界面。需要在保持设备侧统一访问 7458 的前提下，建立独立于 GVA 通用附件系统的日志采集、对象存储和审计能力。

## What Changes

- 在 TR-069 监听端口 7458 增加 `PUT/POST /acs/log`，兼容厂商设备使用尾斜杠、PUT 路径追加文件名和 POST multipart `file` 字段，并为以后同样支持 PUT/POST 的 `/acs/pm`、`/acs/mr` 预留可注册文件通道；所有上传形式共用流式文件处理链，均不进入 CWMP XML 解析和 RawDump。
- 使用全局 LOG 凭据 Profile 校验所有基站上传，支持 HTTP Basic 和 Digest；用户名和密码同时为空时以无认证模式运行，同时非空时启用认证，只配置一项时拒绝保存。
- 共享账号只证明请求可以访问 LOG 通道，不作为设备身份。设备身份由已注册设备、最近 Inform 建立的来源 IP 绑定，以及唯一等待文件的主动 Upload 任务共同解析；无法唯一解析或设备未注册时拒绝上传。
- 保持 `Device.LogMgmt.*` 由用户在基站侧自行配置；GVA 不自动读取、映射或下发这些参数，本阶段不引入 TR-181 适配。
- 新增流式文件接收、大小限制、SHA-256、并发限制、超时、幂等和中断清理，不将文件内容载入内存或写入日志。
- 新增独立的传输任务、文件制品和事件记录；文件实际存储使用抽象存储接口，首个生产实现使用 MinIO/S3 兼容 API。
- 主动 Upload RPC 与周期上传都形成可审计的 LOG 任务；主动 RPC 固定使用 `/acs/log` 且 Username/Password 为空，由基站使用本地 LOG 凭据，周期上传根据接收到的文件自动建档。
- 在 GVA 的 TR-069 菜单下新增“日志文件”页面，使用数据库自增文件 ID，支持分页、按设备序列号精确过滤和权限受控的日志下载；文件列表不展示内部状态、SHA-256、OUI 或数据库设备 ID。

## Capabilities

### New Capabilities

- `tr069-file-ingress`: 规定单端口文件通道、共享认证、设备身份解析、流式接收、主动/周期任务关联、状态和可靠性行为。
- `tr069-log-artifact-management`: 规定日志制品持久化、MinIO 存储抽象、查询、按设备 ID 过滤、下载、权限和审计行为。

### Modified Capabilities

- 无。

## Impact

- GVA 后端：TR-069 7458 路由、中间件、Inform 设备绑定、Upload/TransferComplete 关联、配置、模型、仓储、服务、管理 API 和权限注册。
- GVA 前端：TR-069 菜单、日志文件查询页、设备序列号筛选、自增文件 ID 和下载操作。
- 数据库：新增传输任务、文件制品和传输事件表，不复用 GVA 通用附件表。
- Redis：新增有短 TTL 的来源 IP 到已注册设备绑定，以及必要的上传并发/唤醒状态；MySQL 仍是任务和制品元数据的事实来源。
- 对象存储：新增中立的制品存储接口和 MinIO/S3 兼容实现；不暴露对象键、存储凭据或本地路径。
- 部署：LOG 凭据由全局凭据服务加密保存，文件通道仍使用 YAML 配置 MinIO、限制和固定路径，并要求为 7458 文件流量配置合适的代理超时与请求体限制。
- 测试：Basic/Digest 认证、设备解析、未注册/歧义拒绝、流式限长、任务状态、MinIO 失败恢复、API 权限、设备 ID 过滤和前端主题兼容。

```

## openspec/changes/add-tr069-log-collection/design.md

- Source: openspec/changes/add-tr069-log-collection/design.md
- Lines: 1-207
- SHA256: 7b946610b0b6cdff719f92249c79a35004a94a13e2794902c0b2cd3e0ac4c2c5

[TRUNCATED]

```md
## Context

当前 TR-069 插件在独立监听端口 `7458` 上处理 `POST /acs`。CWMP RawDump 中间件会读取整个请求体并记录 XML，这种处理适合受控大小的 SOAP 报文，但不能用于约 20 MiB 的压缩日志文件。GVA 已有附件上传和 MinIO 支持，但现有适配器以 `multipart.FileHeader` 和内存缓冲为中心，无法满足设备以 HTTP PUT/POST 原始请求体流式上传、传输中止清理和跨存储迁移的要求。

TR-069 的 Upload RPC 由 ACS 下发给设备，包含目标 URL、Username、Password 和 CommandKey。确认后的厂商扩展契约要求 GVA 下发固定目标 URL，但 Username/Password 始终为空，由 BS 使用用户在 `Device.LogMgmt.*` 中预配置的本地 LOG 凭据访问文件入口。标准文件传输使用 HTTP PUT；为兼容 BS 厂商实现，本项目同时接受 POST。后续文件请求无论使用 PUT 还是 POST，都不包含标准 DeviceIdStruct，也不保证回传 CommandKey。所有 BS 共用全局 LOG Profile，因而认证账号只能标识“允许访问 LOG 通道”，不能标识具体设备。设备身份必须由先前成功认证的 Inform 保存的设备标识和来源 IP 唯一解析。

相关参与者包括 BS/CPE、TR-069 7458 文件入口、TR-069 命令管理器、MySQL、Redis、MinIO，以及通过 GVA JWT/Casbin 访问日志页面的管理用户。

## Goals / Non-Goals

**Goals:**

- 在同一个 7458 监听端口上隔离 CWMP XML 和大文件上传处理路径。
- 使用全局 LOG Profile 实现 Basic/Digest 认证；Profile 同时为空时允许无认证，同时非空时强制认证，半配置状态被凭据服务拒绝。
- 只接受能够唯一映射到已注册设备的上传，不使用用户名作为设备身份。
- 支持主动 Upload 与设备周期上传，保留任务、事件、文件和失败原因。
- 以固定内存流式写入 MinIO，校验大小和 SHA-256，并可恢复跨 MySQL/对象存储的不一致状态。
- 在 TR-069 菜单增加“日志文件”，使用自增文件 ID，支持按设备序列号精确查询和受权限保护的下载。
- 通过存储接口和通道注册机制为以后 S3/Ceph、PM/MR 文件类型保留扩展点。

**Non-Goals:**

- 不自动读取、映射、下发或维护 `Device.LogMgmt.*`；这些参数由用户在 BS 上配置。
- 不适配 TR-181，也不解析压缩包内部日志内容。
- 第一阶段不启用 `/acs/pm`、`/acs/mr`，不实现 PM/MR 业务模型或页面。
- 不复用 GVA 通用附件表和基于 multipart 的上传接口。
- 不在第一阶段支持 NAT 后多设备共用同一可见源 IP；无法唯一识别时拒绝，而不是猜测设备。
- 不把 MinIO 对象键、凭据或预签名地址作为永久 API 数据返回给前端。

## Decisions

### 1. 单端口、分路由、分中间件

设备统一访问 7458：

| 方法与路由 | 用途 | 中间件链 |
| --- | --- | --- |
| `POST /acs` | Inform、RPC Response、TransferComplete | CWMP RawDump、XML 解析、CWMP Handler |
| `PUT/POST /acs/log[/*filename]` | 压缩日志上传 | 文件认证、设备解析、并发限制、流式接收 |
| `PUT/POST /acs/pm` | 预留 | 默认未注册/禁用 |
| `PUT/POST /acs/mr` | 预留 | 默认未注册/禁用 |

文件通道固定允许 PUT 和 POST，二者共享同一 handler。标准 raw PUT/POST 直接把请求体视为文件字节流；为兼容当前 BS 厂商脚本，还接受尾斜杠、`PUT /acs/log/<filename>` 和 `POST /acs/log/` 的 multipart `file` 字段。multipart 要求第一个 part 就是 `file`，使用 `MultipartReader` 只流式读取该 part，不调用 `ParseMultipartForm`、不产生 `multipart.FileHeader`、临时文件或整包缓冲；此前放置其他字段会立即拒绝，避免在并发准入前排空大字段。其他方法返回 `405 Method Not Allowed` 并通过 `Allow: PUT, POST` 声明支持范围。文件名后缀只允许 PUT 使用单个路径段。文件路由绝不经过 RawDump、XML 解析或 `io.ReadAll`。选择同端口可简化 BS、防火墙和部署配置；路由隔离仍可让 SOAP 与文件流量使用不同限制。替代方案是独立文件端口，但会增加设备配置和网络暴露，并不能消除存储和认证复杂度。

### 2. 通道配置和启动校验

TR-069 配置保留 `file-ingress` 和 `artifact-store` 的路径、限制和存储设置；用户名和密码改由 `secure-tr069-credential-authentication` 提供的全局 LOG Profile 动态解析：

```yaml
tr069:
  address: ":7458"
  file-ingress:
    enabled: true
    public-base-url: "http://host.docker.internal:7458"
    trusted-proxies: []
    identity-binding-ttl: 30m
    channels:
      log:
        enabled: true
        path: /acs/log
        max-file-size: 67108864
        max-concurrent: 4
        max-concurrent-per-device: 1
        upload-timeout: 10m
        retention-days: 30
        storage-prefix: log
  artifact-store:
    driver: minio
    endpoint: "127.0.0.1:9000"
    bucket: "gva-tr069"
    access-key: "${TR069_MINIO_ACCESS_KEY}"
    secret-key: "${TR069_MINIO_SECRET_KEY}"
    use-ssl: false
    prefix: artifacts
```

当文件入口或 LOG 通道启用时，路径、存储驱动和必要的 MinIO 配置必须合法，否则 TR-069 插件启动失败。LOG Profile 为空不再阻止启动，而是明确启用无认证模式；半配置状态无法保存。默认文件上限 64 MiB，为当前约 20 MiB 日志留出余量；所有限制均可配置。配置中的秘密不写入普通日志、GVA 操作日志或 API 响应。

### 3. 共享账号只认证通道


```

Full source: openspec/changes/add-tr069-log-collection/design.md

## openspec/changes/add-tr069-log-collection/tasks.md

- Source: openspec/changes/add-tr069-log-collection/tasks.md
- Lines: 1-92
- SHA256: 9ac7e999c141cc15d77acbd06ef79f9b65ebf92736f012ea861175d04886eb93

[TRUNCATED]

```md
## 1. 配置与数据基础

- [x] 1.1 为启用LOG入口但凭据为空、通道限制非法、可信代理非法和MinIO配置不完整添加失败测试
- [x] 1.2 增加类型化文件入口、通道、Basic/Digest认证、可信代理、设备绑定和制品存储配置及启动校验
- [x] 1.3 为传输任务、制品、事件默认值、索引、状态约束和无秘密序列化添加失败测试
- [x] 1.4 实现三种TR-069传输模型并注册插件AutoMigrate
- [x] 1.5 实现事务创建、条件状态变更、设备过滤分页、陈旧接收扫描和事件追加仓储

## 2. Inform设备绑定与来源解析

- [x] 2.1 测试直连IP、可信/不可信代理、IPv4/IPv6规范化、绑定过期和来源IP歧义
- [x] 2.2 抽取CWMP与文件入口共用的可信代理客户端IP解析器
- [x] 2.3 仅在Inform设备持久化成功后刷新Redis绑定，并保存设备完整身份和TTL
- [x] 2.4 实现只返回唯一近期注册设备的MySQL回退及Redis回填
- [x] 2.5 实现主动任务、来源IP和设备身份解析以及统一外部拒绝与详细内部事件

## 3. Basic与Digest认证

- [x] 3.1 添加PUT/POST协议向量测试，覆盖Basic、Digest、MD5、MD5-sess、过期nonce、重放、缺失凭据和空配置
- [x] 3.2 实现共享LOG Basic常量时间认证和安全审计字段
- [x] 3.3 实现Digest challenge、校验、过期nonce和Redis防重放
- [x] 3.4 组合路由认证，使成功认证只授权配置通道且不从用户名推导设备

## 4. 流式制品存储

- [x] 4.1 定义存储中立的ArtifactStore、ArtifactWriter、对象元数据和类型化错误
- [x] 4.2 添加Begin/Write/Commit/Abort/Open/Stat/Delete、取消、重复Abort和对象不可用契约测试
- [x] 4.3 实现MinIO/S3兼容流式multipart驱动，禁止整文件缓存
- [x] 4.4 为服务和路由测试提供隔离内存或临时目录存储
- [x] 4.5 实现确定性不透明对象键和安全原始文件名元数据，DTO不暴露存储字段

## 5. LOG文件入口流水线

- [x] 5.1 测试PUT/POST `/acs/log`绕过RawDump/XML、PM/MR禁用隔离和其他方法405行为
- [x] 5.2 实现通道注册并为每个启用文件路由挂载统一的路由专用中间件和处理链
- [x] 5.3 实现全局、通道、设备准入控制，返回503与Retry-After并保证释放令牌
- [x] 5.4 实现接收元数据、固定缓冲流式传输、大小限制、SHA-256、超时、取消Abort和成功响应
- [x] 5.5 实现同内容幂等、异内容冲突拒绝和无秘密/无正文结构化事件
- [x] 5.6 验证认证与身份失败均发生在对象创建前，拒绝请求不产生可用制品
- [x] 5.7 添加真实BS文件名后缀、尾斜杠multipart、基础路径认证、不重定向、忙拒绝和201响应回归
- [x] 5.8 实现真实BS路由、认证、文件名、multipart流式和状态码兼容且不改变CWMP `/acs`

## 6. 主动Upload与传输状态机

- [x] 6.1 测试ACTIVE/PERIODIC分类、单一等待任务、响应/文件/TransferComplete乱序、故障、超时和歧义
- [x] 6.2 现有Upload命令被接受时事务创建ACTIVE任务，并保留URL、凭据和CommandKey关联
- [x] 6.3 把文件关联到唯一WAITING_FILE任务；没有等待任务时创建PERIODIC任务
- [x] 6.4 实现UploadResponse状态0/1、成功TransferComplete乱序和故障保留制品的幂等完成
- [x] 6.5 集成等待文件与TransferComplete超时，使所有任务进入明确终态

## 7. 协调与保留期

- [x] 7.1 测试对象已提交但数据库未完成、无对象陈旧接收、校验和不匹配、重复协调和删除失败
- [x] 7.2 实现陈旧RECEIVING协调器，完成匹配对象或清理残留并标记失败
- [x] 7.3 实现条件DELETING/DELETED保留期清理并保存任务和事件审计
- [x] 7.4 注册Worker生命周期并保证多服务实例不会重复处理同一行

## 8. 管理API与权限

- [ ] 8.1 添加分页、精确设备ID、数据范围、不可用制品、缺失对象、JWT/Casbin拒绝和秘密字段省略测试
- [x] 8.2 实现带设备身份和精确deviceId过滤的制品列表DTO、服务和API
- [ ] 8.3 实现安全Content-Disposition、取消、设备权限复核和操作审计的后端流式下载
- [x] 8.4 在TR-069插件初始化器注册列表/下载API和Casbin权限
- [x] 8.5 将制品UUID替换为自增文件ID并更新模型、事件、生命周期、对象键、审计和下载查找

## 9. GVA日志文件页面

- [ ] 9.1 添加制品分页、设备ID参数、鉴权二进制下载和后端错误处理的前端API封装测试
- [x] 9.2 增加TR-069“日志文件”菜单和路由，位于告警子菜单之前且不改变既有路由标识
- [x] 9.3 构建支持设备ID搜索、重置、分页、时间、身份、文件名、大小、校验和、来源、状态和条件下载的GVA页面
- [ ] 9.4 添加设备过滤、分页、不可用下载、权限和明暗主题token组件测试
- [x] 9.5 使用仓库生成器更新前端路由组件元数据并验证生产构建解析
- [x] 9.6 精简列表为文件ID、SerialNumber、文件名、来源、可读大小、时间和下载，并使用完整SerialNumber过滤

## 10. 集成、安全与运维

- [x] 10.1 增加MinIO和文件入口示例配置、秘密注入、桶初始化、健康检查和Docker开发接线
- [ ] 10.2 添加PUT/POST Basic/Digest端到端客户端，覆盖Inform绑定、流式日志、MinIO元数据、幂等、过滤和下载
- [x] 10.3 运行现有TR-069 CWMP、核心、后端和前端回归并验证POST `/acs`不变
- [ ] 10.4 用户配置`Device.LogMgmt.*`后测试真实BS周期上传及主动Upload/TransferComplete

```

Full source: openspec/changes/add-tr069-log-collection/tasks.md

## openspec/changes/add-tr069-log-collection/specs/tr069-file-ingress/spec.md

- Source: openspec/changes/add-tr069-log-collection/specs/tr069-file-ingress/spec.md
- Lines: 1-170
- SHA256: aee71d02b712acd48a898690ae2e74cd26455d40689f218487f919af3d0403df

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: Shared LOG channel authentication
The system SHALL resolve the global LOG credential Profile for every `PUT /acs/log` and `POST /acs/log` request, SHALL support HTTP Basic and Digest for both methods when both credential fields are configured, and SHALL operate without HTTP authentication when both fields are empty.

#### Scenario: Both LOG credential fields are empty
- **WHEN** the LOG file ingress channel is enabled and the LOG Profile username and password are both empty
- **THEN** the system SHALL keep the channel available without requiring Authorization while retaining device resolution and resource protection

#### Scenario: LOG credential Profile is invalid
- **WHEN** an attempted credential update provides only username or only password
- **THEN** the credential service SHALL reject the update and SHALL preserve the previous atomic Profile

#### Scenario: Request has no authentication
- **WHEN** a client sends `PUT /acs/log` or `POST /acs/log` without valid Basic or Digest authentication
- **THEN** the system SHALL return `401` with an authentication challenge and SHALL not read or persist the file body

#### Scenario: Shared credentials are valid
- **WHEN** a client sends valid configured Basic or Digest credentials with either supported upload method
- **THEN** the system SHALL authorize access to the LOG channel without treating the username as a device identifier

### Requirement: PUT and POST upload compatibility
The system SHALL accept both PUT and POST for every enabled file-ingress channel, SHALL accept the configured base path with or without a trailing slash, SHALL accept a single filename appended to a PUT path, and SHALL process every accepted form through the same authentication, identity, admission, streaming, storage, and state pipeline. Raw PUT/POST bodies SHALL be treated as artifact bytes, while a POST `multipart/form-data` compatibility envelope SHALL require its first part to be the `file` part and SHALL stream only that part without buffering the whole request.

#### Scenario: Device uploads with PUT
- **WHEN** an authenticated and uniquely resolved device sends `PUT /acs/log` with a raw file body
- **THEN** the system SHALL process the file through the LOG ingress pipeline

#### Scenario: Vendor device appends a filename to PUT
- **WHEN** an authenticated and uniquely resolved device sends `PUT /acs/log/<filename>` with a raw file body
- **THEN** the system SHALL stream the raw body and SHALL preserve a sanitized filename as artifact metadata without using it as an object key

#### Scenario: Device uploads with POST
- **WHEN** an authenticated and uniquely resolved device sends `POST /acs/log` with a raw file body
- **THEN** the system SHALL process the file identically to PUT without requiring multipart form data

#### Scenario: Vendor device falls back to multipart POST
- **WHEN** an authenticated and uniquely resolved device sends `POST /acs/log/` whose first multipart field is `file`
- **THEN** the system SHALL stream only that file part through the LOG ingress pipeline without redirecting, buffering the whole request, or persisting multipart boundaries

#### Scenario: Unsupported method is used
- **WHEN** a client uses a method other than PUT or POST on an enabled file-ingress path
- **THEN** the system SHALL return `405` with `Allow: PUT, POST` and SHALL not read or persist an artifact body

### Requirement: Unique registered-device resolution
The system SHALL resolve an authenticated upload to exactly one registered device using its trusted source IP, recent Inform identity binding, and any unique active Upload task, and SHALL reject requests that cannot be resolved uniquely.

#### Scenario: Recent Inform uniquely binds the source IP
- **WHEN** an authenticated upload arrives from a source IP that has one valid recent Inform binding to a registered device
- **THEN** the system SHALL associate the upload with that device ID and SHALL record the OUI, ProductClass, and SerialNumber identity used for verification

#### Scenario: Multiple devices match the source IP
- **WHEN** more than one registered device or active task can match the authenticated request's source IP
- **THEN** the system SHALL reject the upload with a generic `403` response and SHALL internally record `DEVICE_AMBIGUOUS`

#### Scenario: Device is unknown or binding expired
- **WHEN** no currently registered device can be uniquely resolved within the configured identity-binding window
- **THEN** the system SHALL reject the upload with a generic `403` response and SHALL not create an available artifact

#### Scenario: Forwarded address comes from an untrusted peer
- **WHEN** a request contains forwarded-IP headers but its direct peer is not in the configured trusted-proxy ranges
- **THEN** the system SHALL ignore those headers and use the direct peer address for device resolution

### Requirement: Inform refreshes upload identity binding
The system SHALL refresh a TTL-bound source-IP-to-device binding only after a valid Inform has been accepted and the device identity has been persisted.

#### Scenario: Registered device sends Inform
- **WHEN** a valid Inform for a registered or newly registered device is successfully persisted
- **THEN** the system SHALL store a binding containing device ID, OUI, ProductClass, SerialNumber, normalized source IP, and Inform time for the configured TTL

#### Scenario: Invalid Inform is rejected
- **WHEN** an Inform fails protocol validation or device persistence
- **THEN** the system SHALL NOT create or refresh a file-ingress identity binding

### Requirement: Route-specific body handling
The system SHALL process CWMP XML and file-upload routes through separate middleware chains, and SHALL never send a file body through RawDump or XML parsing.

#### Scenario: Log file is uploaded
- **WHEN** an authenticated and resolved device sends `PUT /acs/log` or `POST /acs/log`
- **THEN** the request body SHALL be streamed directly to the artifact store with bounded buffering and SHALL not be copied into XML logs, traces, or database fields

```

Full source: openspec/changes/add-tr069-log-collection/specs/tr069-file-ingress/spec.md

## openspec/changes/add-tr069-log-collection/specs/tr069-log-artifact-management/spec.md

- Source: openspec/changes/add-tr069-log-collection/specs/tr069-log-artifact-management/spec.md
- Lines: 1-103
- SHA256: 2bfbda3e202bc9c8bb67fc1580c6c25ac768e7192751924a05a724604358de91

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: Independent transfer metadata
The system SHALL persist TR-069 transfer tasks, artifacts, and transfer events in plugin-owned tables rather than the GVA generic attachment table.

Each artifact SHALL use a database-generated unsigned integer `fileId` as its sole file identifier. The system SHALL NOT create or expose a separate artifact UUID. Transfer task and command identifiers remain independent protocol and workflow identifiers.

#### Scenario: Artifact record is created
- **WHEN** an upload is admitted and its `RECEIVING` artifact metadata is inserted
- **THEN** MySQL SHALL allocate a non-reusable auto-increment `fileId` and subsequent storage, events, reconciliation, audit, query, and download operations SHALL reference that numeric file ID

#### Scenario: Upload receive begins
- **WHEN** an authenticated upload has been uniquely resolved and admitted
- **THEN** the system SHALL create auditable task/artifact metadata in `RECEIVING` state before making a file downloadable

#### Scenario: Upload fails
- **WHEN** storage, checksum, size, timeout, or client transport fails
- **THEN** the system SHALL retain a failure event and terminal or recoverable state without exposing a partial file for download

### Requirement: Storage-provider abstraction
The system SHALL write and read artifact bytes through a streaming store abstraction whose production implementation supports MinIO/S3-compatible object storage.

#### Scenario: MinIO upload succeeds
- **WHEN** the file stream is committed successfully to the configured MinIO bucket and database metadata is updated
- **THEN** the artifact SHALL become `AVAILABLE` with an opaque service-generated object key, size, and SHA-256

#### Scenario: Object storage driver changes
- **WHEN** a future deployment selects another S3-compatible service such as Ceph RGW
- **THEN** transfer authentication, task state, management APIs, and the GVA page SHALL not require provider-specific changes

### Requirement: Cross-store reconciliation
The system SHALL reconcile incomplete MySQL/object-storage operations and SHALL expose only artifacts confirmed available in both metadata and storage.

#### Scenario: Object commit succeeds but metadata finalization is interrupted
- **WHEN** a stale `RECEIVING` record has a matching object with expected metadata
- **THEN** the reconciler SHALL idempotently finalize the artifact as `AVAILABLE`

#### Scenario: Stale receive has no valid object
- **WHEN** a `RECEIVING` record exceeds its timeout and no matching complete object exists
- **THEN** the reconciler SHALL abort or delete residual storage and mark the artifact failed

### Requirement: Log artifact query API
The system SHALL provide a JWT/Casbin-protected paginated management API that exposes only `AVAILABLE` LOG artifacts and supports exact filtering by the device SerialNumber business identifier.

#### Scenario: User filters by device ID
- **WHEN** an authorized user requests the artifact list with a complete alphanumeric `serialNumber`
- **THEN** the system SHALL use an equality filter and return only available LOG artifacts belonging to that SerialNumber and permitted by the user's device data scope

#### Scenario: User does not provide device ID
- **WHEN** an authorized user requests the artifact list without a SerialNumber
- **THEN** the system SHALL return a paginated list limited to devices within that user's data scope

#### Scenario: Unauthorized list access
- **WHEN** a user lacks the artifact-list permission
- **THEN** the system SHALL deny the request without revealing artifact metadata

### Requirement: Protected artifact download
The system SHALL stream an available artifact to an authorized GVA user only after revalidating JWT, Casbin, device data permission, and artifact state.

#### Scenario: Authorized user downloads available artifact
- **WHEN** a user with access to the artifact's device requests its numeric `fileId` and the artifact is `AVAILABLE`
- **THEN** the backend SHALL stream the object with a safe filename and SHALL record a GVA operation audit entry

#### Scenario: User lacks device access
- **WHEN** a user can call the endpoint but lacks data permission for the artifact's device
- **THEN** the system SHALL deny download without exposing the MinIO object key or storage credentials

#### Scenario: Artifact is not available
- **WHEN** an artifact is receiving, failed, deleting, deleted, or missing in object storage
- **THEN** the system SHALL not return a file body and SHALL return an appropriate non-success response

### Requirement: TR-069 log files menu
The system SHALL add a “日志文件” page under the TR-069 menu that follows the active GVA light/dark theme and exposes device-filtered query and download operations.

#### Scenario: User opens the page
- **WHEN** an authorized user opens the TR-069 “日志文件” menu
- **THEN** the page SHALL show a GVA-styled search area, paginated table, text device-ID filter backed by SerialNumber, numeric file ID, filename, source, human-readable size, receive time, and permitted download action

#### Scenario: User changes GVA theme
- **WHEN** the application switches between supported light and dark themes

```

Full source: openspec/changes/add-tr069-log-collection/specs/tr069-log-artifact-management/spec.md
