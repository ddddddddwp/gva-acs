# TR-069 基站日志采集

## 入口与边界

TR-069 ACS 与文件上传共用 `7458` 端口，但使用独立路由和中间件：

| 方法与路由 | 用途 | 默认状态 |
| --- | --- | --- |
| `POST /acs` | CWMP Inform、RPC Response、TransferComplete | 启用 |
| `PUT/POST /acs/log` | LOG 原始文件流上传 | 由 `fileIngress` 控制 |
| `PUT/POST /acs/pm` | PM 文件流上传 | 禁用 |
| `PUT/POST /acs/mr` | MR 文件流上传 | 禁用 |

文件入口不接受 multipart，不进入 CWMP XML 解析和 RawDump。请求体必须直接是压缩日志字节；服务端以固定缓冲流式写入对象存储，并在传输过程中计算 SHA-256。

## GVA 配置

默认配置是安全关闭状态。启用前必须配置非空的共享用户名和密码、可被基站访问的公网/管理网 URL，以及 MinIO 凭据：

```yaml
tr069:
  fileIngress:
    enabled: true
    publicBaseURL: http://172.17.0.1:7458
    trustedProxies: []
    identityBindingTTL: 1800
    authentication:
      username: replace-with-log-user
      password: replace-with-a-strong-secret
      schemes: [digest, basic]
      realm: GVA-TR069-LOG
      nonceTTL: 300
    channels:
      log:
        enabled: true
        path: /acs/log
        maxFileSize: 67108864
        maxConcurrent: 4
        maxConcurrentPerDevice: 1
        uploadTimeout: 600
        retentionDays: 30
        storagePrefix: log
    artifactStore:
      driver: minio
      endpoint: 127.0.0.1:19000
      bucket: gva-tr069-artifacts
      accessKey: replace-with-minio-user
      secretKey: replace-with-minio-secret
      useSSL: false
      prefix: artifacts
```

配置中的密码不会通过配置查询接口输出，也不得写入操作日志。仓库只保留空凭据示例；部署凭据应由环境专用配置或密钥管理系统注入。开启 `fileIngress` 时缺少任一必要字段会阻止 TR-069 插件启动，热加载到无效配置时继续使用上一个有效快照。

`publicBaseURL` 是下发 Upload RPC 时使用的基础地址。容器内 BS 常用 Docker 网桥宿主地址（例如 `172.17.0.1`）；跨主机部署必须替换为基站实际可达的管理网地址。

## 基站配置

GVA 不会自动设置 `Device.LogMgmt.*`。周期上传由用户在基站侧配置：

```text
Device.LogMgmt.URL      = http://<GVA可达地址>:7458/acs/log
Device.LogMgmt.Username = <tr069.fileIngress.authentication.username>
Device.LogMgmt.Password = <tr069.fileIngress.authentication.password>
```

周期、日志级别和启用参数按设备现有 `Device.LogMgmt.*` 能力配置。本功能当前只适配项目已有 TR-069 数据模型，不引入 TR-181 参数映射。

文件传输支持 HTTP Basic 与 Digest（MD5、MD5-sess、`qop=auth`）。Basic 会直接携带可还原的凭据，Digest 也不加密文件内容，因此生产环境必须在反向代理或入口网关启用 HTTPS。

## 设备识别

共享上传账号仅用于认证通道，不能作为设备身份。基站必须先通过 `POST /acs` 成功发送 Inform；GVA 在设备落库后，将来源地址与设备 ID、OUI、ProductClass、SerialNumber 建立短期绑定。上传时按以下顺序解析：

1. 使用可信代理规则得到真实来源 IP；默认不信任转发头。
2. 查询仍在有效期内的 Inform 绑定。
3. 必要时使用数据库中近期注册设备回退并回填缓存。
4. 只有唯一匹配且已注册的设备才能上传。

未认证返回 `401` 并携带允许的认证挑战；未注册、绑定过期或无法识别返回 `403`；同一来源匹配多台设备也返回 `403`，不会猜测设备身份。多个基站经同一 NAT 出口时来源 IP 不唯一，这是共享账号方案的已知限制。此场景应使用设备可区分的管理地址、可信网关注入的真实地址，或后续增加设备级凭据/签名标识。

## 主动采集与周期上传

- 主动采集：GVA 创建 Upload 命令和 ACTIVE 传输任务，使用当前共享 URL/账号/密码构造 RPC；文件、UploadResponse 和 TransferComplete 可乱序到达，状态机按事实幂等归并。
- 周期上传：没有等待中的 ACTIVE 任务时，成功接收文件会创建 PERIODIC 任务；不强制要求 TransferComplete，以对象已提交且校验完成为完成事实。

所有 Upload RPC 的持久化参数使用凭据占位符，实际发送时才从当前有效配置注入明文。上传中断、超限和对象存储失败都会中止临时对象并记录不含文件内容和凭据的事件。

常见文件入口响应：

| 状态码 | 含义 |
| --- | --- |
| `204` | 文件接收、校验和对象提交成功；重复的相同内容也幂等成功 |
| `401` | Basic/Digest 缺失、错误、过期或 Digest 重放 |
| `403` | 设备未完成 Inform 注册、身份过期或来源匹配不唯一 |
| `405` | 非 PUT/POST 方法，响应包含 `Allow: PUT, POST` |
| `409` | 同一任务重复上传了不同内容或任务关联冲突 |
| `413` | Content-Length 或流式接收大小超过通道上限 |
| `503` | 全局、通道或设备并发已满，按 `Retry-After` 重试 |

## MinIO、下载与保留策略

开发 Compose 提供：

- S3 API：`127.0.0.1:19000`
- Console：`http://127.0.0.1:19001`
- 容器内端点：`minio:9000`
- 默认 bucket：`gva-tr069-artifacts`

MinIO 容器使用 `restart: "no"`，不会随 Docker/Windows 自动拉起。根账号来自 `GVA_MINIO_ROOT_USER` 和 `GVA_MINIO_ROOT_PASSWORD`；Compose 默认值仅供隔离开发环境，使用前应修改。GVA 启动文件入口时会检查并创建缺失 bucket。

用户从 GVA 管理端“TR069管理 → 日志文件”按精确设备 ID 查询，并通过受 JWT/Casbin 保护的 `/tr069/artifact/:artifactId/download` 流式下载。API 不返回对象键、驱动、来源 IP 或凭据；下载审计只记录用户、设备 ID、制品 ID、结果和耗时，不缓存文件响应体。

到达 `retentionDays` 后，后台任务先将记录条件更新为 `DELETING`，删除对象成功后再标记 `DELETED`。删除失败保留记录并重试，任务、校验和及事件时间线继续用于审计。

## 迁移到 Ceph 或其他 S3

业务层只依赖 `ArtifactStore`/`ArtifactWriter` 的流式接口。迁移到 Ceph RGW 或其他 S3 兼容服务时，优先复用 MinIO SDK 驱动并修改 endpoint、TLS 和凭据；非 S3 后端只需新增存储适配器，不应修改路由、认证、任务表或页面 DTO。迁移步骤应先停止新上传，复制对象并校验 SHA-256，再切换 endpoint，最后恢复入口和保留任务。
