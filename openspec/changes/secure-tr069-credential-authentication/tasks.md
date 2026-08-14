## 1. 凭据数据基础

- [ ] 1.1 为四个全局Profile、五个认证通道映射、设备通道状态和审计模型编写失败测试
- [ ] 1.2 实现模型、唯一索引、种子数据和TR-069插件AutoMigrate注册
- [ ] 1.3 实现凭据加密仓储、主密钥校验、set/clear原子语义和revision递增
- [ ] 1.4 实现通道到Profile解析、状态失效和不含密码的DTO/API服务

## 2. 共享HTTP认证核心

- [ ] 2.1 为Basic、Digest MD5/MD5-sess、qop=auth、nonce过期/重放、空Profile编写协议向量测试
- [ ] 2.2 抽取通道无关CredentialProvider、Authenticator和认证Principal
- [ ] 2.3 实现Basic常量时间比较与Digest challenge/校验，复用Redis nonce store
- [ ] 2.4 实现失败审计、正常challenge区分、Authorization和错误脱敏

## 3. CWMP设备接入认证

- [ ] 3.1 添加失败测试，证明401请求不会进入CWMP Handler、解析Inform或创建设备
- [ ] 3.2 在7458 CWMP路由前挂载接入认证中间件并传递安全context
- [ ] 3.3 在Inform成功持久化后更新CWMP_ACCESS状态，失败只记录来源IP审计
- [ ] 3.4 验证空Connection Profile保持现有无认证CWMP行为

## 4. Connection Request改造

- [ ] 4.1 添加测试，证明缺失设备凭据不再生成`gva-connection-request` SPV命令
- [ ] 4.2 删除/禁用自动Provisioner调度入口并为废弃配置输出安全警告
- [ ] 4.3 让Connection Request客户端通过通道Provider读取全局Connection凭据并兼容Basic/Digest
- [ ] 4.4 分别记录HTTP认证和后续Connection Request Inform结果
- [ ] 4.5 将旧AUTO Profile标记LEGACY/INACTIVE并验证运行时不再读取旧密文

## 5. 管理API与前端

- [ ] 5.1 实现四Profile查询、原子设置、显式清除API及JWT/Casbin权限
- [ ] 5.2 为半配置拒绝、密码不回显、权限和revision状态失效添加API测试
- [ ] 5.3 实现四卡凭据管理页面，包含Connection双通道共享说明和显式清除确认
- [ ] 5.4 在设备详情展示五通道状态、最近成功与不可信安全事件，不增加手动校验按钮
- [ ] 5.5 添加前端API、表单校验、密码处理和状态展示测试

## 6. 安全、迁移与验收

- [ ] 6.1 扩展XML、命令、Trace、RawDump和操作日志脱敏测试，覆盖通用Username/Password与Authorization
- [ ] 6.2 更新配置、部署文档和数据库迁移说明，明确主密钥、HTTPS和全局凭据风险
- [ ] 6.3 运行TR-069后端、前端、race/static回归并确认无明文秘密
- [ ] 6.4 使用两台Docker BS验证正确/错误/空Connection接入与已有设备保留行为
- [ ] 6.5 验证实际Connection Request使用共享凭据且数据库不产生新的凭据下发命令
