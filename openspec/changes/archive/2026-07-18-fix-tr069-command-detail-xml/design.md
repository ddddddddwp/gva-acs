# 修复方案

## 命令详情

在 `CommandStore.Detail` 读取命令后，仅为返回值复制并合并展示参数：当命令存在 `CommandKey` 时，将其写入返回对象的 `params.commandKey`。数据库中的 `params_json` 不更新，避免重复事实来源。

## XML 关联

为 `CommandXMLSink` 增加命令解析能力。事件已有 `CommandID` 时保持现有路径；缺少时且存在 `CWMPID`，通过带索引的 `tr069_commands.request_id` 查询对应 `command_id`。解析成功后保存 XML；未找到时继续忽略非命令协议流量，数据库错误写安全日志。

该方式同时覆盖：

- 出站 RPC：`MarkSending` 已先保存 `request_id`，随后 Builder 生成 XML。
- 入站响应：响应 CWMP ID 与原请求一致，可以关联同一命令。

## 测试

- Service：断言详情合并 `commandKey` 且不修改数据库原始参数。
- Adapter：断言缺少 CommandID 的出站和入站事件均可按 CWMPID 落库；无匹配事件不产生记录。
