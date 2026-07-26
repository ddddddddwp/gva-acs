---
comet_change: add-tr069-log-collection
role: technical-design
canonical_spec: openspec
---

# TR-069 LOG凭据流程修订技术设计

## 保留的实现

保留固定`/acs/log`路由、Basic/Digest认证器、Inform/IP唯一设备解析、流式对象存储、传输任务/制品/事件、ACTIVE/PERIODIC状态机、协调器、保留期、列表与下载基础。此次不重写已验证的大文件路径。

## 认证Provider替换

`RuntimeFileCredentialProvider`改为通道Provider适配器，通过LOG_UPLOAD解析全局LOG Profile。文件入口配置只保留路径、realm/nonce策略、限制与存储。Profile关闭时认证中间件返回disabled principal并继续设备解析；Profile启用时行为与现有协议向量一致。

## 主动Upload命令

`CommandApi.Upload`只要求文件入口、publicBaseURL和LOG通道可用，不再要求服务器凭据非空。提交payload固定为LOG FileType、`<publicBaseURL>/acs/log`、空Username、空Password及DelaySeconds。删除`LogUploadPayloadCodec`对秘密的hydrate责任；若保留codec，只允许规范化URL和断言凭据为空。

命令存储、XML生成和RawDump测试必须证明没有文件凭据。BS收到命令后使用本地`Device.LogMgmt.Username/Password`完成后续HTTP认证，这是两台模拟基站的验收前置条件。

## 任务关联

主动命令继续事务创建唯一WAITING_FILE任务。来自相同设备IP的文件优先关联唯一主动任务，否则创建PERIODIC。空RPC凭据不改变CommandKey、UploadResponse或TransferComplete状态机。

## 测试与收尾

先添加API/codec失败测试，再替换Provider和payload。完成原change未完成的列表/下载/前端测试和端到端上传；最后对两台BS分别运行主动空凭据与周期本地凭据测试，并扫描数据库和日志秘密。
