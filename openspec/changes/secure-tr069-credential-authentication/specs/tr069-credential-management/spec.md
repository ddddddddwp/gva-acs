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

### Requirement: 全局凭据管理界面
系统 SHALL 提供四个 Profile 的管理界面和设备五通道状态展示，不得提供脱离真实协议动作的人工密码校验。

#### Scenario: 修改凭据
- **WHEN** 管理员在页面修改用户名
- **THEN** 页面要求同时输入新密码并通过原子保存 API提交

#### Scenario: 查看设备状态
- **WHEN** 管理员打开设备详情
- **THEN** 页面分别展示CWMP接入、Connection Request、LOG、PM、MR的启用/待验证/成功状态和最近安全事件
