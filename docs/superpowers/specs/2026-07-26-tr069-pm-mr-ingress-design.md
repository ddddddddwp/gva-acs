---
comet_change: enable-tr069-pm-mr-ingress
role: technical-design
canonical_spec: openspec
---

# TR-069 PM/MR固定入口技术设计

## 通道参数化

扩展现有FileIngress channel policy而不是复制LOG Handler。PM、MR分别配置path、maxFileSize、并发、timeout、retentionDays、storagePrefix和auth channel。路由注册器为每个enabled channel挂载同一认证、设备解析、准入和接收链。

固定路径为`/acs/pm`、`/acs/mr`，兼容现有文件名后缀与multipart规则。每个路由解析出的channel必须和认证principal及存储policy一致，禁止用户输入任意channel绕过。

## 认证与设备归属

PM_UPLOAD和MR_UPLOAD通过凭据服务取得独立Profile。认证失败在读取文件前返回401；关闭认证时继续业务。认证通过后使用规范化RemoteAddr、可信代理规则、Redis Inform绑定和MySQL近期回退解析唯一设备。歧义/过期/未知均403。

## 周期任务与制品

PM/MR接收不查询ACTIVE任务，直接创建PERIODIC任务。任务和Artifact使用channel索引，确定性对象键分别落入`pm/`和`mr/`前缀。保存成功即完成；AutonomousTransferComplete仅作为可选事件。协调、清理和幂等逻辑按channel参数化。

## 管理面

后端列表和下载API强制channel白名单，复核JWT、Casbin、设备数据范围和AVAILABLE状态。前端复用文件表格/下载组件，但提供独立PM、MR菜单与权限，且不渲染立即上传按钮。

## 测试顺序

1. 参数化路由测试覆盖PM/MR raw PUT、POST、multipart和非法路径。
2. 独立/交叉/空Profile Basic/Digest测试。
3. 唯一IP、无绑定、过期和NAT歧义测试。
4. PERIODIC-only、对象前缀、限长、并发、超时、协调和清理测试。
5. API/Casbin/Vue channel隔离测试。
6. 两台BS人工配置固定URL与本地凭据后的周期上传黑盒验收。

## 部署顺序

必须先部署凭据核心和修订LOG，再启用PM/MR。初始路径配置可存在但保持disabled；设置GVA Profile和基站本地参数后逐台启用。回滚只关闭PM/MR channel，不删除制品或影响LOG/CWMP。
