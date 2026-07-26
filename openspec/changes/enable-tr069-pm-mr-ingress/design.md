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
