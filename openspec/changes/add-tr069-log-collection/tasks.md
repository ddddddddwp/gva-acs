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
- [ ] 10.5 运行race/static并验证64MiB限长上传不会产生整文件分配或正文日志
- [x] 10.6 文档化共享凭据/NAT限制、HTTPS、失败码、恢复、保留期、轮换和Ceph/S3迁移
- [x] 10.7 重置开发传输元数据和MinIO LOG对象，重启GVA并验证真实BS上传、自增ID和SerialNumber下载

## 11. 已确认的凭据流程修订

- [ ] 11.1 添加空LOG Profile允许无认证入口和半配置原子拒绝的失败测试
- [ ] 11.2 用全局LOG_UPLOAD CredentialProvider替换YAML运行时凭据，同时保持Basic/Digest和固定`/acs/log`
- [ ] 11.3 添加主动LOG Upload持久化及传输空Username/Password和固定URL的失败命令测试
- [ ] 11.4 删除LOG命令秘密hydrate并确保命令JSON/XML、RawDump、Trace和操作日志不含文件凭据
- [ ] 11.5 验证两台Docker BS在空凭据Upload RPC后使用本地LOG凭据并完成ACTIVE任务
- [ ] 11.6 重新验证两台BS周期LOG上传、唯一IP归属和凭据revision状态更新
