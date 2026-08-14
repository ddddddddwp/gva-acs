## ADDED Requirements

### Requirement: PM与MR固定文件入口
系统 SHALL 在TR-069的7458监听器上提供`PUT/POST /acs/pm[/*filename]`和`PUT/POST /acs/mr[/*filename]`，且文件请求不得进入CWMP XML解析或RawDump。

#### Scenario: PM raw PUT
- **WHEN** 基站向`/acs/pm`或单段文件名后缀发送合法PUT文件流
- **THEN** 系统通过PM通道流式处理并返回成功状态

#### Scenario: MR multipart POST
- **WHEN** 基站向`/acs/mr`发送兼容的multipart `file` POST
- **THEN** 系统通过MR通道流式读取文件且不创建整包内存或临时文件

#### Scenario: 非法方法或路径
- **WHEN** 请求使用不支持的方法、多段文件后缀或跨通道路由
- **THEN** 系统返回405或404且不创建任务或对象

### Requirement: PM与MR独立凭据
PM入口 MUST 只使用PM Profile，MR入口 MUST 只使用MR Profile，并自动兼容Basic/Digest或对应空凭据无认证模式。

#### Scenario: 正确PM凭据
- **WHEN** 请求使用当前PM凭据访问`/acs/pm`
- **THEN** 认证授予PM_UPLOAD principal并继续设备解析

#### Scenario: 跨模块凭据
- **WHEN** 请求使用LOG或MR凭据访问`/acs/pm`
- **THEN** 系统返回401且不读取文件正文、不创建对象

#### Scenario: PM认证关闭
- **WHEN** PM用户名和密码都为空
- **THEN** `/acs/pm`保持业务可用且不要求Authorization

### Requirement: 唯一已注册设备解析
系统 MUST 在认证成功后根据最近成功Inform的来源IP绑定解析唯一已注册设备。

#### Scenario: 唯一匹配
- **WHEN** 上传来源IP在有效窗口内唯一映射到一个已注册设备
- **THEN** 文件归属该设备并继续接收

#### Scenario: 无匹配或绑定过期
- **WHEN** 来源IP没有有效设备绑定
- **THEN** 系统返回403且不创建可用制品

#### Scenario: NAT歧义
- **WHEN** 来源IP映射到多个候选设备
- **THEN** 系统返回403并记录DEVICE_AMBIGUOUS事件，不猜测归属

### Requirement: PM与MR仅周期任务
系统 SHALL 把成功接收的PM和MR文件分别记录为channel=PM/MR、source=PERIODIC，不得为两类通道创建主动Upload命令。

#### Scenario: PM周期上传
- **WHEN** PM文件成功保存并校验
- **THEN** 系统创建或完成PERIODIC PM任务及AVAILABLE制品

#### Scenario: 禁止主动上传
- **WHEN** 管理用户查看PM或MR文件页面/API
- **THEN** 系统不提供立即上传操作或对应Upload命令端点

### Requirement: 流式资源保护与一致性
PM/MR SHALL 复用文件通道的限长、并发、超时、SHA-256、幂等、Abort、协调和保留期行为。

#### Scenario: 文件超限
- **WHEN** 声明长度或流式读取超过通道上限
- **THEN** 系统返回413、终止对象上传并且不产生AVAILABLE制品

#### Scenario: 客户端中断
- **WHEN** 上传在完成前断开或超时
- **THEN** 系统释放并发资源、Abort存储写入并记录FAILED任务

### Requirement: PM与MR管理权限隔离
系统 SHALL 提供按channel过滤的PM/MR列表和下载，并再次校验用户权限、设备数据范围和制品AVAILABLE状态。

#### Scenario: PM列表
- **WHEN** 有权限用户查询PM文件
- **THEN** 只返回其设备数据范围内的AVAILABLE PM制品，不返回MR/LOG或存储秘密

#### Scenario: 越权下载
- **WHEN** 用户尝试下载无数据权限设备或不同channel的文件
- **THEN** 系统拒绝请求且不返回对象内容或对象键

### Requirement: GVA不管理基站PM/MR参数
系统 MUST NOT 读取后自动复制、设置或下发基站PM/MR的URL、用户名、密码、启用和周期参数。

#### Scenario: 启用PM/MR入口
- **WHEN** 管理员在GVA配置PM或MR全局凭据
- **THEN** 系统只更新GVA认证Profile，不创建任何设备SetParameterValues命令
