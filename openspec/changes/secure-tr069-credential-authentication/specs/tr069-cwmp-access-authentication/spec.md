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
