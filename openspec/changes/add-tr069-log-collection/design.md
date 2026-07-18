## Context

当前 TR-069 插件在独立监听端口 `7458` 上处理 `POST /acs`。CWMP RawDump 中间件会读取整个请求体并记录 XML，这种处理适合受控大小的 SOAP 报文，但不能用于约 20 MiB 的压缩日志文件。GVA 已有附件上传和 MinIO 支持，但现有适配器以 `multipart.FileHeader` 和内存缓冲为中心，无法满足设备以 HTTP PUT 流式上传、传输中止清理和跨存储迁移的要求。

TR-069 的 Upload RPC 由 ACS 下发给设备，包含目标 URL、Username、Password 和 CommandKey；随后设备发起的 HTTP PUT 不包含标准 DeviceIdStruct，也不保证回传 CommandKey。项目决定所有 BS 共用一套配置文件中的 LOG 账号，因而认证账号只能标识“允许访问 LOG 通道”，不能标识具体设备。设备身份必须由先前 Inform 保存的设备标识和来源 IP 唯一解析。

相关参与者包括 BS/CPE、TR-069 7458 文件入口、TR-069 命令管理器、MySQL、Redis、MinIO，以及通过 GVA JWT/Casbin 访问日志页面的管理用户。

## Goals / Non-Goals

**Goals:**

- 在同一个 7458 监听端口上隔离 CWMP XML 和大文件上传处理路径。
- 使用共享长期账号实现 Basic/Digest 认证，并拒绝空配置、空凭据和错误凭据。
- 只接受能够唯一映射到已注册设备的上传，不使用用户名作为设备身份。
- 支持主动 Upload 与设备周期上传，保留任务、事件、文件和失败原因。
- 以固定内存流式写入 MinIO，校验大小和 SHA-256，并可恢复跨 MySQL/对象存储的不一致状态。
- 在 TR-069 菜单增加“日志文件”，支持按设备 ID 查询和受权限保护的下载。
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
| `PUT /acs/log` | 压缩日志上传 | 文件认证、设备解析、并发限制、流式接收 |
| `PUT /acs/pm` | 预留 | 默认未注册/禁用 |
| `PUT /acs/mr` | 预留 | 默认未注册/禁用 |

文件路由绝不经过 RawDump、XML 解析或 `io.ReadAll`。选择同端口可简化 BS、防火墙和部署配置；路由隔离仍可让 SOAP 与文件流量使用不同限制。替代方案是独立文件端口，但会增加设备配置和网络暴露，并不能消除存储和认证复杂度。

### 2. 通道配置和启动校验

TR-069 配置新增 `file-ingress` 和 `artifact-store`：

```yaml
tr069:
  address: ":7458"
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

当文件入口或 LOG 通道启用时，共享用户名、密码、realm、存储驱动和必要的 MinIO 配置必须非空且合法，否则 TR-069 插件启动失败并给出不含秘密的配置错误。默认文件上限 64 MiB，为当前约 20 MiB 日志留出余量；所有限制均可配置。配置中的秘密不写入普通日志、GVA 操作日志或 API 响应。

### 3. 共享账号只认证通道

`PUT /acs/log` 支持 HTTP Basic 和 Digest。Digest 支持 `qop=auth`，并兼容设备常见的 MD5/MD5-sess；算法和挑战行为通过协议测试固定。Digest nonce 有有效期并校验 nonce-count，重放状态存入 Redis。Basic 与 Digest 使用同一套共享账号，生产环境仍应通过 HTTPS 保护文件正文和 Basic 凭据。

认证失败返回 `401` 和适当的 `WWW-Authenticate` 挑战。认证成功只得到 `channel=LOG`，不会从 username 推导 deviceId。相比每设备账号，共享账号降低 BS 配置成本，但失去凭据级设备隔离；该权衡由当前部署规模和明确需求接受。

### 4. Inform/IP 唯一设备解析

每次成功处理 Inform 时，现有设备入库继续保存 `OUI`、`ProductClass`、`SerialNumber`、来源 IP 和 `LastInform`，并在 Redis 写入短期绑定：

```text
tr069:file-ingress:ip:<normalized-ip>
  -> {deviceId, OUI, ProductClass, SerialNumber, informedAt}
```

默认 TTL 为 30 分钟，由配置调整。设备 ID 以数据库内部设备记录为准，协议身份使用 OUI、ProductClass、SerialNumber 交叉校验。

来源 IP 的确定规则是：只有 HTTP 直连对端属于 `trusted-proxies` 时才信任 `X-Forwarded-For`/`X-Real-IP`，否则只使用 `RemoteAddr`。文件到达时按以下顺序解析：

1. 查找同一来源 IP、同一 LOG 通道且处于 `WAITING_FILE` 的唯一主动 Upload 任务；
2. 查找有效期内最近 Inform 写入的唯一 Redis 绑定；
3. 必要时查询 MySQL 中同一规范化 IP 且 `LastInform` 仍在绑定窗口内的唯一已注册设备，并回填 Redis；
4. 交叉校验设备未删除、协议身份完整且仍为已注册状态。

任一步出现多个候选、候选设备身份不一致、绑定过期或没有候选，都以同一个对外 `403` 响应拒绝；内部事件区分 `DEVICE_NOT_RESOLVED`、`DEVICE_AMBIGUOUS` 和 `DEVICE_NOT_REGISTERED`。不通过错误正文泄露设备是否存在。

这种设计接受“当前环境来源 IP 可唯一对应设备”的约束。若以后多个 BS 位于同一 NAT 后，应升级为每设备凭据、URL 中的一次性不透明 token 或设备支持的额外可信请求头，而不是放宽唯一性校验。

### 5. 主动与周期上传的关联

主动采集继续通过现有 Upload RPC 创建 GVA 命令，同时创建 `source=ACTIVE` 的传输任务并进入 `WAITING_FILE`。RPC 中的 URL 指向 `/acs/log`，用户名和密码来自共享配置，CommandKey 继续关联 UploadResponse 和 TransferComplete。设备发起 PUT 时，如果该设备只有一个等待文件的主动 LOG 任务，则文件关联该任务；否则没有主动任务时自动创建 `source=PERIODIC` 的任务。

同一设备同一时间只允许一个 LOG 文件流和一个等待文件的主动任务，避免共享 URL 不携带任务标识造成错误关联。若周期文件恰好与主动等待窗口重叠，协议本身无法从 PUT 区分两者，系统优先关联唯一主动任务并在事件中记录推断来源。这是共享 URL/共享账号方案的明确限制。

主动任务完成规则：

- UploadResponse `Status=0`：文件已完整保存并校验后完成。
- UploadResponse `Status=1`：文件已保存且对应 TransferComplete 成功后完成，二者可任意顺序到达。
- TransferComplete 失败：任务失败但保留已接收文件。
- 超时：沿用可配置的 TransferComplete 超时；等待文件和接收文件使用文件通道自己的超时。

周期任务以文件成功保存为完成事实，不要求标准 TransferComplete；若设备另发 AutonomousTransferComplete，只追加事件和关联信息。

### 6. 流式接收和资源保护

接收顺序为：认证、设备解析、获取全局/通道/设备并发令牌、创建 `RECEIVING` 记录、开始对象写入、限长复制并同步计算 SHA-256、提交对象、条件更新数据库为 `AVAILABLE`、更新任务、返回 `204 No Content`。

- 请求 `Content-Length` 已超过限制时立即返回 `413`；未知长度使用 `MaxBytesReader`/限长 reader 在流中强制上限。
- 使用固定大小缓冲池，文件内容不进入 Zap、RawDump、Trace 或数据库。
- 全局或设备并发已满返回 `503` 和 `Retry-After`。
- 请求上下文取消、上传超时、大小超限或客户端断开都会 Abort multipart upload 并把制品标记为失败。
- 同一任务重复上传相同 SHA-256 和大小时返回幂等成功；内容不同则新增冲突事件并拒绝覆盖。
- 日志只记录 deviceId、taskId、artifactId、大小、SHA-256、耗时、来源 IP 摘要和失败阶段。

### 7. 独立制品存储接口和一致性

插件定义不依赖 MinIO SDK 的接口：

```text
ArtifactStore.Begin(ctx, objectKey, metadata) -> ArtifactWriter
ArtifactWriter.Write(p)
ArtifactWriter.Commit(ctx) -> StoredObject
ArtifactWriter.Abort(ctx)
ArtifactStore.Open(ctx, objectKey) -> ReadCloser + Stat
ArtifactStore.Stat(ctx, objectKey)
ArtifactStore.Delete(ctx, objectKey)
```

第一阶段生产驱动为 MinIO/S3 兼容实现，测试提供内存或临时目录实现。Ceph RGW 可通过相同 S3 驱动接入；若未来使用原生 Ceph，只新增 Store 实现。对象键由服务端生成：`<prefix>/log/<device-id>/<YYYY>/<MM>/<DD>/<artifact-id>`，原始文件名只作为经过清洗的元数据保存。

MySQL 和对象存储无法形成单事务，数据库使用以下状态处理一致性：

- `RECEIVING`：元数据已创建，multipart upload 未完成；
- `AVAILABLE`：对象提交完成并通过大小/SHA-256 校验，可查询和下载；
- `FAILED`：接收或提交失败，不可下载但保留审计信息；
- `DELETING/DELETED`：保留期清理状态。

后台协调器扫描超时的 `RECEIVING`：若对象存在且元数据匹配则补记 `AVAILABLE`，否则 Abort/删除残留并标记 `FAILED`。下载只允许 `AVAILABLE`。清理任务先条件更新为 `DELETING`，删除对象后改为 `DELETED`，失败可重试。

### 8. 数据模型

新增三张独立表：

- `tr069_transfer_tasks`：UUID、device_id、channel、source、关联 command_id/command_key、状态、UploadResponse 状态、TransferComplete 状态、错误码、时间戳。
- `tr069_artifacts`：UUID、task_id、device_id、channel、状态、storage_driver、object_key、原始文件名、content_type、size、sha256、source_ip、received_at、删除时间。
- `tr069_transfer_events`：task_id、artifact_id、事件码、阶段、前后状态、非敏感消息和结构化元数据、时间戳。

`device_id + received_at`、`task_id`、`state + updated_at`、`sha256` 建立必要索引。共享认证来自配置文件，因此不创建 `tr069_upload_credentials`；GVA 不管理 `Device.LogMgmt.*`，因此不创建日志策略表。

### 9. 管理 API、菜单和下载

TR-069 一级菜单下新增“日志文件”，路由 `logFiles`，组件 `plugin/tr069/view/log-file/index.vue`。页面使用 GVA 的 `gva-search-box`、`gva-table-box`、Element Plus 主题变量和响应式布局，不写死亮色/暗色。

首版查询接口：

```text
GET /tr069/artifact/list?page=1&pageSize=10&deviceId=<id>
GET /tr069/artifact/:artifactId/download
```

列表支持分页和设备 ID 精确过滤，返回接收时间、设备 ID、SerialNumber/OUI、原始文件名、大小、SHA-256、来源、状态和可下载标识。后端始终再次校验 JWT、Casbin、设备数据权限和 `AVAILABLE` 状态，不能依赖前端隐藏按钮。下载由 GVA 后端从 Store 流式转发并写入操作审计；第一阶段不直接向前端暴露 MinIO 预签名地址。

### 10. 状态与事实来源

MySQL 是任务、事件和制品元数据的事实来源。Redis 只保存有 TTL 的 IP 设备绑定、Digest 防重放和并发辅助状态，丢失后可从最近 Inform/MySQL 安全重建或拒绝上传，不会导致制品记录消失。MinIO 是文件字节事实来源，但对象只有在 MySQL 状态为 `AVAILABLE` 时对管理用户可见。

## Risks / Trade-offs

- [共享账号泄漏会允许攻击者访问 LOG 上传入口] → 配置密钥不入库/日志，支持轮换，生产使用 HTTPS，并仍要求来源 IP 唯一映射到已注册设备。
- [NAT 或代理导致多个设备共用来源 IP] → 可信代理白名单、唯一性查询和歧义拒绝；以后引入每设备凭据或一次性 token。
- [周期上传与主动任务同时发生而被错误关联] → 每设备并发为 1、只允许一个主动等待任务、优先唯一主动任务并记录推断事件。
- [MySQL 与 MinIO 部分成功] → 明确制品状态、确定性对象键、幂等条件更新和后台协调器。
- [大文件耗尽连接、内存或磁盘] → 流式 multipart、64 MiB 默认限长、固定缓冲、并发令牌、超时和中断 Abort。
- [Basic/Digest 在纯 HTTP 下不能保护文件正文] → 同时支持设备兼容方案，但生产部署要求 HTTPS；Digest nonce 防重放不能替代传输加密。
- [完整 CWMP XML 可能包含共享密码] → 不改变现有完整 XML 策略，但为包含文件凭据的 XML 详情配置更严格权限，并建议后续单独评审加密存储。

## Migration Plan

1. 停止 TR-069 插件文件入口，部署数据库模型、MinIO bucket/凭据和配置校验。
2. 运行 AutoMigrate 创建三张新表，初始化日志菜单、API 与 Casbin 权限。
3. 启动后端，先验证 `/acs` CWMP 回归，再用测试客户端验证 Basic/Digest、限长和 MinIO。
4. 在单台 BS 上由用户配置 `Device.LogMgmt.URL/Username/Password`，验证 Inform 后周期上传和按设备 ID 查询下载。
5. 验证主动 Upload、UploadResponse、文件 PUT 和 TransferComplete 的乱序组合。
6. 回滚时先禁用 `file-ingress` 并保留表和对象；旧 CWMP `/acs` 不依赖新通道，可独立继续运行。

## Open Questions

无。当前范围已明确采用共享 LOG 凭据、Inform/IP 唯一解析、MinIO 首发存储和按设备 ID 查询下载；NAT 多设备、TR-181 和 PM/MR 留待后续变更。
