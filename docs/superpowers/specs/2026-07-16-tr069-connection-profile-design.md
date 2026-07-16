# TR-069 Connection Profile 自动接入设计

## 背景

GVA 当前已经能从部分 Inform 参数中读取
`Device.ManagementServer.ConnectionRequestURL`，但该能力分散在设备仓储和参数值表中：

- Inform 未携带 URL 时，现有设备更新可能把已经保存的 URL 清空。
- GetParameterValues 会保存 ManagementServer 参数，却不会把 URL 同步回设备的主动唤醒入口。
- 每次 Connection Request 分别查询设备表和参数值表，并为请求创建新的 HTTP Client。
- 现有实现只允许 HTTPS 且只支持 Basic Authentication，无法兼容 TR-069 的 HTTP Digest Connection Request 和当前 BS 的 HTTP 地址。
- 用户名、密码没有独立生命周期，无法区分“已发现”“正在配置”“可用”和“配置失败”。

本设计新增独立的 Connection Profile 插件组件，统一管理设备主动唤醒地址、凭据和配置状态，同时保持 CWMP core 与 GVA 数据库解耦。

## 目标

1. 新设备接入时自动发现并持久化 Connection Request URL。
2. 自动为每台设备生成独立的 Connection Request 用户名和密码，并通过 CWMP `SetParameterValues` 配置到 CPE。
3. 后续主动 RPC 优先使用持久化 Profile 唤醒设备，再复用现有命令队列发送请求。
4. 支持人工覆盖 URL 和凭据，人工配置优先于自动发现，直到用户清除覆盖。
5. 密码加密保存；线上实际发送内容不变，但日志、命令参数和持久化 XML 中必须脱敏。
6. 采集过程复用既有解析结果，不重复解析 XML，不给 Inform 主路径增加明显开销。
7. Connection Request 使用共享连接池和 HTTP Digest Authentication，并兼容当前 BS 的 HTTP 地址。

## 非目标

- 不把 Connection Profile 放入独立微服务。
- 不改变 `tr069-core-only` 对 GVA 数据库、Redis 和权限系统零依赖的边界。
- 不用 Redis 作为 Profile 的事实来源。
- 不在本次实现 UDP Connection Request、XMPP Connection Request 或 STUN。
- 不在前端展示或提供已保存密码的明文回读。

## 架构

Connection Profile 作为 TR-069 adapter 层的独立组件，由四个接口组成：

1. `ConnectionProfileCollector`
   观察已经解析完成的 Inform/GPV 参数，只处理三个精确参数名。
2. `ConnectionProfileRepository`
   负责 Profile 的事务写入、查询、人工覆盖和状态条件更新。
3. `ConnectionCredentialProvisioner`
   生成每设备凭据，并通过现有 RPC 编排器创建系统级 `SetParameterValues` 命令。
4. `ConnectionRequestResolver`
   为主动 RPC 返回有效 URL、用户名和解密后的密码，并执行 HTTP Digest 唤醒。

Collector 通过 adapter hook/decorator 接入 `core.DeviceRepo` 和 GPV 持久化路径，不使用 Gin 全局中间件或 GORM 回调。这样可以保留设备、会话、参数来源和命令关联上下文，又不会让 core 依赖 GVA 模型。

数据库仍是 Profile 和命令状态的事实来源。Redis 继续只承担现有设备命令顺序、锁和唤醒信号。

## 数据模型

新增 `tr069_connection_request_profiles`，每台设备最多一条记录，`device_id` 建唯一索引。

建议字段：

| 字段 | 含义 |
|---|---|
| `device_id` | 关联 TR-069 设备，唯一 |
| `discovered_url` | Inform/GPV 最近发现的 URL |
| `override_url` | 人工覆盖 URL；为空时使用 discovered URL |
| `username` | 当前生效用户名 |
| `password_ciphertext` | AEAD 加密后的密码，不保存明文 |
| `credential_key_version` | 加密密钥版本，支持轮换 |
| `credential_source` | `AUTO` 或 `MANUAL` |
| `auth_scheme` | 当前为 `DIGEST` |
| `provision_state` | `DISCOVERED`、`PROVISIONING`、`READY`、`FAILED` |
| `provision_command_id` | 自动配置凭据所关联的 RPC 命令 |
| `last_error` | 最近一次配置或解析错误 |
| `last_wake_at` | 最近一次 Connection Request 时间 |
| `last_wake_status` | 最近一次唤醒结果 |
| `created_at` / `updated_at` | 审计时间 |

有效 URL 的选择顺序为 `override_url`、`discovered_url`。人工凭据保持最高优先级，自动采集不得覆盖，除非用户明确清除人工覆盖。

`tr069_devices.connection_req_url` 暂时保留为兼容镜像字段，设备列表和旧代码不会立即失效；Connection Request 执行以新 Profile 为准。

## 接入与自动配置流程

### 1. Inform/GPV 采集

Collector 从已经解析的参数映射中检查：

- `Device.ManagementServer.ConnectionRequestURL`
- `Device.ManagementServer.ConnectionRequestUsername`
- `Device.ManagementServer.ConnectionRequestPassword`

采集规则：

- Inform 没有 URL 时不覆盖已保存值。
- URL 非空且规范化后发生变化时才更新 Profile。
- GPV 返回 URL 时使用相同规则补充或更新 Profile。
- ConnectionRequestPassword 读回为空是正常协议行为，不得用空值覆盖加密密码。
- 从设备读取到的用户名只作为发现信息；没有已知密码时不能直接进入 `READY`。
- 人工覆盖存在时，自动发现只更新 discovered 字段，不改变有效配置。

### 2. 自动凭据配置

新设备已有有效 URL、但没有可用凭据时：

1. 生成设备唯一用户名和高强度随机密码。
2. 使用配置中的 AEAD 密钥加密密码并写入 Profile。
3. 在同一设备的现有 RPC FIFO 中创建来源为 `SYSTEM` 的“配置连接请求凭据”命令。
4. 命令构造标准 `SetParameterValues`，同时设置 Username 和 Password。
5. Profile 以命令 ID 进入 `PROVISIONING`。
6. 收到成功响应后使用条件更新进入 `READY`。
7. CWMP Fault、构造失败、发送失败或超时后进入 `FAILED`，保留错误和命令时间线。

系统命令显示在 RPC 记录中，便于排查，但请求参数和 XML 中的 Password 必须脱敏。重复 Inform 不得重复生成正在执行或已经成功的配置命令；只有 Profile 缺少凭据、配置失败后显式重试，或人工请求轮换时才能重新创建。

### 3. 主动 RPC 唤醒

用户提交主动 RPC 后：

1. 如果设备已有可发送的 CWMP 会话，直接进入现有命令构造和发送流程，不执行 Connection Request。
2. 如果没有可用会话，Resolver 使用 `device_id` 一次查询有效 Profile。
3. Profile 非 `READY`、URL 缺失或凭据不可解密时，命令记录明确的 `WAKE_FAILED` 原因并保持既有等待/超时语义。
4. Resolver 对 URL 做安全校验，使用共享 HTTP Client 发起 Digest Connection Request。
5. CPE 返回成功状态后建立新的 Inform 会话，队首命令继续发送。
6. 唤醒结果同时写入 Profile 摘要和命令事件，数据库命令记录仍是执行进度的事实来源。

Connection Request 成功只表示 CPE 接受唤醒请求，不表示 RPC 已完成。

## URL 与 HTTP 安全策略

- 必须支持 `http`；可保留 `https` 作为厂商兼容扩展。
- 默认禁止 URL userinfo，用户名和密码从 Profile 独立加载。
- 禁止自动跟随重定向。
- 设置独立的连接、响应和总请求超时。
- 限制响应体读取大小并立即关闭响应体。
- 可配置允许的目标 CIDR；解析后的目标 IP 必须落在允许范围内，防止设备上报恶意 URL 形成 SSRF。
- 本地开发允许显式加入 `127.0.0.0/8` 和 Docker/实验网段；生产环境由部署配置收紧。
- 复用全局 `http.Transport` 和 `http.Client`，避免每次唤醒新建连接池。
- Digest challenge、nonce、realm、qop 和响应计算必须使用标准实现并覆盖测试，不能退化成只发送 Basic Auth。

## 凭据加密与配置

GVA `tr069:` 配置节点新增 Connection Request 子配置，字段命名在实施阶段与现有配置风格对齐，至少包含：

```yaml
tr069:
  connectionRequest:
    autoProvisionCredentials: true
    credentialKeyVersion: v1
    credentialEncryptionKey: ""
    requestTimeout: 10s
    allowedCIDRs:
      - 127.0.0.0/8
      - 172.16.0.0/12
```

- 仓库配置模板中的密钥保持为空。
- 本地密钥只放入已忽略的 `server/config.local.yaml` 或受控环境配置。
- 开启自动配置但缺少有效密钥时，插件必须明确报告配置错误，不得退化为明文存储。
- 使用 AES-256-GCM 或等价 AEAD；每次加密使用独立随机 nonce。
- API、日志和错误信息不得输出密钥、解密密码或密文。
- 密钥轮换通过 `credential_key_version` 识别旧数据，允许读取旧版本并逐步重加密。

## 选择性脱敏

脱敏只作用于日志/存储副本，绝不能修改准备发送给 CPE 的 CWMP Message 或线上字节。

至少覆盖：

1. RPC 命令结构化参数。
2. RPC 记录 API 返回值。
3. GVA 操作日志和 core/adapter 调试日志。
4. 完整发送 XML 和接收 XML 的持久化副本。

XML 脱敏器应解析 `ParameterValueStruct`，当 Name 精确等于
`Device.ManagementServer.ConnectionRequestPassword` 时，把对应 Value 替换成固定标记 `******`。不得使用可能误伤其他节点的全局字符串替换。

脱敏器失败时采用 fail-closed：不写入可能含密码的 XML，并记录不包含原始载荷的安全错误。历史上如已存在该参数的原始 XML，实施迁移时执行一次定向清理。

## 性能设计

- Collector 复用 Inform/GPV 已生成的参数 map，只做三个 O(1) 精确键查询。
- 规范化后与当前值相同则不执行 UPDATE。
- `device_id` 唯一索引保证唤醒查询为单行读取。
- 每次唤醒只读取一次 Profile，不再分别查询设备表和参数值表。
- HTTP Client/Transport 全局复用。
- 密码只在 Profile 写入和实际唤醒时进行加解密。
- 第一阶段不增加 Redis 或本地秘密缓存；用真实指标证明单行查询成为瓶颈后再评估短 TTL 缓存。
- 自动配置命令沿用现有设备 FIFO、恢复扫描和幂等机制，不新增第二套任务队列。

## 迁移与兼容

1. AutoMigrate 创建 Profile 表和唯一索引。
2. 从 `tr069_devices.connection_req_url` 回填 discovered URL。
3. 必要时从 `tr069_datamodel_values` 补充最新 URL/Username；空 Password 不回填。
4. 修复设备 Upsert：Inform 缺少 URL 时不得清空旧设备字段。
5. 主动唤醒切换到 Resolver；旧字段只作为迁移期兼容回退，完成验证后再移除回退。
6. 自动凭据功能仅处理具备有效 URL 的设备；已有设备可由后台扫描分批进入 DISCOVERED，不在启动时一次性下发。

## 错误与可观测性

错误按阶段记录，但不包含秘密：

- `profile.collect`：URL 解析或参数采集失败。
- `profile.encrypt`：凭据生成/加密失败。
- `profile.provision`：内部 SPV 创建或执行失败。
- `connection_request.resolve`：Profile 缺失、未就绪或解密失败。
- `connection_request.validate`：协议、主机、端口或 CIDR 不允许。
- `connection_request.digest`：Digest challenge/认证失败。
- `connection_request.http`：连接、超时或非成功状态。

RPC 命令时间线继续显示 `WAKE_FAILED`、`REQUEST_SENT`、`RESPONSE_COMPLETED` 等事件；Profile 只提供设备级最近状态，不取代命令记录。

## 测试与验收

### 后端单元/集成测试

- Inform 缺少 URL 不清空历史 URL。
- Inform 和 GPV 均可发现/更新 URL，未变化时不重复写库。
- 人工覆盖优先且不会被自动发现覆盖。
- Password 空读回不覆盖加密密码。
- 每台设备只创建一个进行中的自动配置命令。
- 自动 SPV 成功、Fault、发送失败和超时正确更新 Profile。
- AES-GCM 往返、错误密钥、密钥版本和 nonce 唯一性测试。
- 命令参数、API、日志和持久化 XML 脱敏；线上发送 XML 保持原文。
- Profile 单次查询和共享 HTTP Client 行为测试。
- HTTP Digest challenge 成功、凭据错误、超时、重定向拒绝和 CIDR 拒绝测试。
- 服务重启后 PROVISIONING 状态可通过现有命令恢复/终态继续收敛。

### BS 联调

- Inform 自动保存 `http://127.0.0.1:8400` 或实际容器可达地址。
- 首次会话自动完成 Username/Password 的 SetParameterValues。
- 设备离线会话状态下提交安全查询类 RPC，GVA 通过 Digest 唤醒设备。
- 新 Inform 到达后 FIFO 队首命令发送并在 RPC 记录中完成。
- 日志和 RPC 页面看不到 ConnectionRequestPassword 明文。

## 验收边界

完成条件：

- Profile 生命周期、自动凭据配置、Digest 唤醒和选择性脱敏均有自动测试。
- 当前 BS 可以使用保存的 URL 和自动配置凭据被成功唤醒。
- 主动命令仍由现有状态机追踪，没有绕过数据库事实来源和设备 FIFO。
- Inform/GPV 主路径没有二次 XML 解析，且不存在每次请求创建 HTTP Transport 的行为。
