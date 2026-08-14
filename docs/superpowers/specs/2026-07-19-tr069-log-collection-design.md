# TR-069 基站日志采集设计

日期：2026-07-19
OpenSpec change：`add-tr069-log-collection`

## 1. 目标

在现有 GVA TR-069 插件中增加独立的基站日志采集能力：BS 继续统一访问 7458，通过 `PUT/POST /acs/log` 流式上传压缩日志；GVA 将文件写入 MinIO，并在 TR-069 菜单提供“日志文件”页面，以自增文件 ID 管理文件并按设备序列号精确查询和下载。

本设计确认以下前提：

- 所有 BS 共用配置文件中的一套长期 LOG 用户名和密码。
- 共享账号只校验是否允许上传，不标识设备。
- 设备由最近 Inform 保存的协议身份和来源 IP 唯一解析。
- 当前部署假设一个可见来源 IP 唯一对应一台设备；不能唯一确定时拒绝上传。
- 用户自行配置 BS 的 `Device.LogMgmt.*`，GVA 不自动配置、不做参数映射。
- 当前只处理 TR-069，不引入 TR-181。
- 文件首先存储到 MinIO，业务层不能依赖 MinIO 具体 API，以便以后迁移到 S3/Ceph。

## 2. 总体架构

```text
BS/CPE
  ├─ POST :7458/acs ──────────────> CWMP RawDump/XML Handler
  │                                  ├─ Inform 更新设备和来源 IP 绑定
  │                                  ├─ UploadResponse
  │                                  └─ TransferComplete
  │
  └─ PUT/POST :7458/acs/log ───────────> File Ingress
                                     ├─ Basic/Digest 共享认证
                                     ├─ 来源 IP -> 已注册设备唯一解析
                                     ├─ 全局/通道/设备并发限制
                                     ├─ 流式 SHA-256 与大小限制
                                     ├─ ArtifactStore -> MinIO
                                     └─ MySQL 任务/文件/事件状态

GVA Web :18080
  └─ TR-069 / 日志文件
       ├─ GET :18888/tr069/artifact/list?deviceId=...
       └─ GET :18888/tr069/artifact/:id/download
```

`POST /acs` 和 `PUT/POST /acs/log` 只共享监听端口，不共享请求体中间件。日志文件不会进入 XML 解析、RawDump、Zap 或 Trace。

## 3. 设备侧路由

| 方法与路由 | 首期行为 |
| --- | --- |
| `POST /acs` | 保持当前 CWMP 会话行为 |
| `PUT/POST /acs/log[/*filename]` | 启用 LOG 文件上传，兼容尾斜杠和 PUT 文件名后缀 |
| `PUT/POST /acs/pm` | 保留通道定义，默认禁用 |
| `PUT/POST /acs/mr` | 保留通道定义，默认禁用 |

URL 不包含设备 ID、任务 ID、用户名或密码：

```text
http://<GVA可达地址>:7458/acs/log
```

TR-069 标准文件传输使用 PUT；为兼容 BS 厂商实现，文件通道同时接受 POST、尾斜杠和单层 PUT 文件名后缀。PUT 和 POST 共用完全相同的认证、设备识别、限流、流式存储和任务状态处理。raw 请求体直接视为压缩文件字节流；POST multipart 要求第一个 part 是 `file`，使用流式 `MultipartReader` 只读取该字段，不缓存整个请求，也不使用 `multipart.FileHeader`。其他 HTTP 方法返回 `405` 和 `Allow: PUT, POST`。

主动 Upload RPC 中的 URL、Username 和 Password 由 GVA 配置提供；周期上传时由用户在 BS 的 `Device.LogMgmt.*` 中配置相同值。

## 4. 认证和设备识别

### 4.1 共享认证

LOG 通道支持 HTTP Basic 和 Digest。启用通道时，配置的 username/password 任一为空都视为启动配置错误；请求缺少凭据、传空值或校验错误都返回 `401`，且在读取文件体之前结束。

Digest 首期兼容 `qop=auth`、MD5 和 MD5-sess，校验时使用请求实际采用的 PUT 或 POST 方法计算摘要；nonce 有过期时间，并用 Redis 校验 nonce-count 防重放。生产必须优先使用 HTTPS，因为 Digest 不加密文件正文，Basic 在纯 HTTP 下也不能保护密码。

### 4.2 文件请求为什么不能直接得到设备 ID

Inform SOAP 中包含 DeviceIdStruct，即 OUI、ProductClass、SerialNumber；但 Upload RPC 是 ACS 下发给设备的请求，后续 HTTP PUT/POST 是独立文件传输，文件请求体不包含 DeviceIdStruct，也不保证带回 CommandKey。因此不能假设“每次文件请求自带设备 ID”，也不能用共享 username 识别设备。

### 4.3 Inform/IP 绑定

Inform 成功入库后，GVA 在 Redis 写入：

```text
tr069:file-ingress:ip:<normalized-ip>
  -> deviceId + OUI + ProductClass + SerialNumber + informedAt
```

默认 TTL 为 30 分钟，可配置。只有设备记录成功写入后才能刷新绑定，非法 Inform 不建立上传资格。

设备解析顺序：

1. 同一来源 IP 是否存在唯一的 `WAITING_FILE` 主动任务；
2. 是否存在有效的最近 Inform Redis 绑定；
3. Redis 丢失时，MySQL 中是否只有一个最近 Inform 且 IP 相同的已注册设备；
4. 交叉验证设备未删除且 OUI、ProductClass、SerialNumber 与保存身份一致。

没有候选、候选超过一个、绑定过期、身份冲突或设备未注册，都对外返回统一 `403`。内部事件分别保存 `DEVICE_NOT_RESOLVED`、`DEVICE_AMBIGUOUS` 或 `DEVICE_NOT_REGISTERED`，避免对外泄露设备信息。

只有直连对端位于 `trusted-proxies` 配置时才读取 `X-Forwarded-For`/`X-Real-IP`，否则使用 socket 对端地址，防止设备伪造来源 IP。

## 5. 主动采集和周期上传

### 5.1 主动采集

```text
用户下发 Upload
  -> 创建 GVA RPC 命令
  -> 同事务创建 ACTIVE 传输任务 WAITING_FILE
  -> 下发 URL + 共享账号 + CommandKey
  -> BS 返回 UploadResponse
  -> BS PUT 或 POST /acs/log
  -> GVA 关联该设备唯一等待任务并存储文件
  -> BS 按协议需要时发送 TransferComplete
  -> 状态机完成或失败
```

完成规则：

- UploadResponse `Status=0`：文件成功存储即完成。
- UploadResponse `Status=1`：文件成功存储并收到成功 TransferComplete 后完成，允许两者乱序。
- TransferComplete 失败：任务失败，已经存储的文件保留用于排查。
- 重复响应、重复 PUT/POST 和重复 TransferComplete 使用条件更新保持幂等。

### 5.2 周期上传

没有等待中的 ACTIVE 任务时，唯一识别出的设备上传文件会自动创建 PERIODIC 任务。周期任务以文件实际接收并校验成功为完成条件，不强制依赖标准 TransferComplete。

### 5.3 已接受的限制

共享 URL 和共享账号使 PUT/POST 文件请求无法天然区分“主动任务文件”与“恰好同时到达的周期文件”。首期通过以下约束降低误关联：

- 每台设备同时最多一个 LOG 文件流；
- 每台设备同时最多一个 `WAITING_FILE` 主动任务；
- 有唯一主动任务时优先关联主动任务，否则建立周期任务；
- 存在多个候选任务时拒绝，不猜测。

若以后出现 NAT 后多设备或更复杂的并发场景，需要另行设计每设备凭据或一次性不透明 token。

## 6. 文件接收和性能

默认上限为 64 MiB，覆盖当前约 20 MiB 的单个压缩包。接收流程：

```text
认证
  -> 设备唯一解析
  -> 获取全局/通道/设备并发令牌
  -> MySQL 创建 RECEIVING 记录
  -> MinIO Begin multipart upload
  -> 固定缓冲流式复制 + SHA-256 + 限长
  -> Commit object
  -> MySQL 条件更新 AVAILABLE
  -> 更新任务和事件
  -> 201 Created
```

必须满足：

- 不使用 `io.ReadAll`、`bytes.Buffer` 或 `multipart.FileHeader` 接收设备日志。
- Content-Length 超限时立即返回 `413`；chunked 传输在 reader 层继续强制限长。
- 每台设备默认并发 1，全局默认并发 4；满载返回 `503` 和 `Retry-After`。
- 默认文件接收超时 10 分钟；TransferComplete 的 12 小时超时继续独立配置。
- 客户端断开、超时或超限必须 Abort multipart upload 并释放令牌。
- 接收过程中只记录 ID、大小、SHA-256、耗时和失败阶段，不记录文件内容或认证密码。
- 同一任务通过 PUT、POST 或两种方法交叉重复上传相同大小和 SHA-256 时幂等成功；内容不同不覆盖原文件。

## 7. 存储和一致性

业务服务只依赖：

```text
ArtifactStore
  ├─ Begin -> ArtifactWriter
  ├─ Open
  ├─ Stat
  └─ Delete

ArtifactWriter
  ├─ Write
  ├─ Commit
  └─ Abort
```

首期实现为 MinIO/S3-compatible Store；测试使用内存或临时目录 Store。对象键由 GVA 生成：

```text
artifacts/log/<device-id>/<YYYY>/<MM>/<DD>/<file-id>
```

原始文件名只作为清洗后的元数据，不能决定对象路径。

MySQL 和 MinIO 不是分布式事务，制品状态分为：

- `RECEIVING`：记录已创建，文件未确认完成；
- `AVAILABLE`：对象和元数据都完成，可下载；
- `FAILED`：接收失败，不可下载；
- `DELETING` / `DELETED`：保留期清理。

后台协调器扫描超时 `RECEIVING`：对象存在且大小/校验匹配则补记 `AVAILABLE`；否则清理残留并标记 `FAILED`。下载永远只读取 `AVAILABLE`。

## 8. 数据模型

不复用 GVA 通用附件表，新增：

| 表 | 主要职责 |
| --- | --- |
| `tr069_transfer_tasks` | 设备、通道、ACTIVE/PERIODIC、CommandKey、状态和超时 |
| `tr069_artifacts` | 自增文件 ID、任务、设备、存储对象、文件名、大小、SHA-256、状态和接收时间；不保留文件 UUID |
| `tr069_transfer_events` | 关联数值文件 ID 的状态时间线、阶段和非敏感错误 |

共享凭据来自配置文件，不创建凭据表。GVA 不管理 `Device.LogMgmt.*`，不创建日志策略表。

## 9. GVA 菜单和 API

TR-069 菜单新增“日志文件”，排在“RPC 记录”之后、“告警管理”之前。页面沿用 GVA 的搜索区、表格、分页和 Element Plus 主题变量，适配亮色/暗色主题。

首版页面功能：

- 使用支持英文的普通文本输入框，按完整 SerialNumber 精确过滤；
- 分页和重置；
- 显示文件 ID、SerialNumber、原始文件名、自动换算为 B/KB/MB/GB 的大小、来源和接收时间；不显示状态、SHA-256、OUI 或数据库设备 ID；
- 对 `AVAILABLE` 文件提供“下载”。

管理 API：

```text
GET /tr069/artifact/list?page=1&pageSize=10&serialNumber=<完整设备序列号>
GET /tr069/artifact/:fileId/download
```

两个接口都经过 JWT、Casbin 和设备数据权限校验。下载由 GVA 后端流式转发并记录操作审计，前端看不到 MinIO 凭据、原始对象键或服务器路径。

## 10. 配置草案

```yaml
tr069:
  file-ingress:
    enabled: true
    public-base-url: "http://host.docker.internal:7458"
    trusted-proxies: []
    identity-binding-ttl: 30m
    authentication:
      username: "${TR069_LOG_USERNAME}"
      password: "${TR069_LOG_PASSWORD}"
      schemes: [digest, basic]
      realm: "GVA-TR069-LOG"
      nonce-ttl: 5m
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

秘密支持通过部署环境注入；日志和 API 不回显实际值。

## 11. 错误码和可观测性

| 情况 | HTTP | 处理 |
| --- | --- | --- |
| 文件通道使用 PUT/POST 以外的方法 | 405 | 返回 `Allow: PUT, POST`，不读取文件 |
| 缺少/错误认证 | 401 | 挑战 Basic/Digest，不读取文件 |
| 设备未注册/无法解析/歧义 | 403 | 外部统一消息，内部记录具体事件 |
| 文件超限 | 413 | Abort，记录大小限制事件 |
| 并发已满 | 503 | 带 `Retry-After` |
| 存储暂时失败 | 503 | 任务保留失败阶段，可重试/协调 |
| 上传成功 | 201 | 无响应正文；兼容当前 BS 只接受 HTTP 200/201 的脚本 |

每个任务和文件都有事件时间线，记录认证后阶段、设备解析、接收开始、对象提交、状态完成和失败原因。共享密码、Authorization、文件正文和 MinIO 密钥禁止进入事件。

## 12. 测试和上线

测试覆盖：

- PUT/POST 的 Basic/Digest 协议向量、Digest 方法摘要、nonce 过期和重放；
- Inform/IP 绑定、可信代理、IPv4/IPv6、未注册和歧义拒绝；
- 64 MiB 限长、chunked 超限、并发、取消、MinIO Abort；
- ACTIVE/PERIODIC 分类、状态 0/1、TransferComplete 乱序、故障和幂等；
- MySQL/MinIO 部分成功后的协调；
- 查询、设备 ID 数据权限、下载和 Casbin；
- 前端亮/暗主题、过滤、分页和下载；
- 现有 `POST /acs`、RPC 命令和设备参数功能回归；
- 单台真实 BS 的周期上传和主动 Upload 联调。

上线顺序：创建 MinIO bucket 和密钥、配置文件入口、AutoMigrate 新表、初始化 API/Casbin/菜单、先回归 CWMP，再配置单台 BS 的 `Device.LogMgmt.*` 灰度验证。回滚只需禁用 `file-ingress`，保留表和对象审计数据，原 `/acs` 不受影响。

## 13. 结论

本设计选择的是“共享通道认证”和“设备唯一解析”分离：共享账号降低 BS 配置成本，Inform/IP 绑定负责设备归属，歧义时安全拒绝。日志文件以固定内存写入 MinIO，MySQL 保留任务和审计真相；GVA 新菜单只通过权限受控 API 查询和下载。该边界能满足当前单设备测试环境，并为以后扩展 PM/MR、S3/Ceph 和更强设备身份机制保留接口。
