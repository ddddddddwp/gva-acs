---
comet_change: secure-tr069-credential-authentication
role: technical-design
canonical_spec: openspec
---

# TR-069全局凭据与Connection认证技术设计

## 组件边界

在`server/plugin/tr069`内部新增四层：模型/仓储负责密文和revision；`CredentialService`负责原子set/clear与通道解析；共享HTTP认证核心负责Basic/Digest；CWMP、文件入口和Connection Request各自提供薄适配。认证核心不得依赖Gin设备注册、传输任务或GORM模型。

建议模型为`CredentialProfile`、`AuthChannelBinding`、`DeviceAuthState`和`AuthAuditEvent`。Profile key及channel使用受控常量；所有唯一约束由数据库保证。连接两个方向通过binding共享同一Profile ID，避免字段复制。

## 保存与读取契约

写API只接受`set{username,password}`或`clear=true`。空字段组合非法，密码不支持通过空字符串隐式保持。仓储在事务中加密、递增revision、更新修改人，并把引用通道状态标为待验证。读API仅返回username、passwordConfigured、revision和聚合状态。

密文沿用版本化AES-GCM封装和部署主密钥；主密钥缺失时允许全空种子数据和读操作，但拒绝非空写入/解密请求。错误不得包含密文长度之外的秘密信息。

## HTTP认证核心

`Authenticate(request, channel)`先解析Profile。关闭认证时返回disabled principal；启用时验证Basic或Digest。首次无Authorization返回typed challenge，不写失败事件。Digest nonce继续使用Redis的一次性/计数消费，校验method、URI、realm、qop、algorithm、cnonce和nc；Basic使用固定长度hash后常量时间比较。

返回principal包含channel、profile ID、revision、scheme，不包含密码。中间件只把principal放入context。审计在统一出口记录result、IP、scheme、channel和错误码；Authorization永不进入结构化字段。

## CWMP入口顺序

路由顺序为请求限制、trace ID、CWMP认证、可选安全RawDump、CWMP Handler。认证失败时不读取body，因此不能信任DeviceIdStruct。认证成功后Handler照常解析；`UpsertFromInform`成功返回设备ID后，由协调服务记录CWMP_ACCESS成功和刷新文件IP绑定。

空Connection Profile保持现有无认证行为。已注册设备的失败请求只产生IP审计，设备的`LastInform`不更新并按现有在线阈值转离线。

## Connection Request改造

删除Inform完成后调度Provisioner的分支和随机凭据生成路径。Repository仍维护discovered/override URL，但运行时用户名密码只来自CONNECTION_REQUEST通道。客户端请求先无Authorization；收到Basic/Digest challenge后按共享核心构造认证。HTTP成功写通道状态，`6 CONNECTION REQUEST` Inform另写回呼事件。

旧Profile增加或派生LEGACY/INACTIVE视图；不得自动删除或解密迁移。废弃配置字段只发一次警告。

## 前后端接口

API提供list、set、clear和设备状态查询；使用独立Casbin资源。前端四卡共享表单组件，Connection卡显示双向映射。密码输入始终为空，修改用户名要求密码；清除有确认。设备详情的五通道状态来自正式状态和安全审计聚合，不提供test按钮。

## 测试顺序

1. SQLite模型/事务/并发revision测试。
2. 固定协议向量和Redis nonce测试。
3. Gin中间件测试证明失败请求不调用下一Handler。
4. 引擎集成测试证明未认证Inform不Upsert。
5. Connection Request fake server Basic/Digest和无SPV测试。
6. API、Casbin、Vue表单与密码回显测试。
7. RawDump/Trace/命令XML秘密扫描。
8. 两台BS正确、错误、空凭据与真实Connection Request黑盒验收。

## 迁移与回滚

先部署空Profile和UI，再停用Provisioner，最后由用户设置基站与GVA共享凭据并启用认证。回滚通过清空Profile恢复无认证入口；旧表保留，绝不恢复本版本期间的自动下发。
