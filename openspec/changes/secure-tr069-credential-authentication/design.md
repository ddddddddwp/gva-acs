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
6. 回滚时关闭新 Profile认证并恢复旧版本程序；新增表保留，不恢复自动下发产生的新命令。

## Open Questions

无。两台 BS 对 Basic/Digest 和空 Upload RPC 本地凭据回退的支持作为构建阶段黑盒验收项，而不是未决产品行为。
