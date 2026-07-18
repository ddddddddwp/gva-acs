# 修复 RPC 命令详情参数与 XML 记录

## 问题

- Reboot 的 `CommandKey` 由服务端单独生成，`params_json` 保持 `{}`，导致详情页“请求参数”不能反映实际下发参数。
- core 已生成并发送完整 XML，但 `wire.xml` 事件缺少 `CommandID` 时被 GVA XML sink 静默过滤，所有命令详情均无 XML 记录。

## 根因

- 命令详情直接返回持久化参数，没有在响应 DTO 中合并独立 `command_key`。
- XML sink 只接受已携带 `CommandID` 的事件，没有利用事件中的 `CWMPID` 与命令表 `request_id` 建立关联。

## 修复目标

- 详情响应合并服务端生成的 `CommandKey`，但不修改原始 `params_json`。
- XML sink 在缺少 `CommandID` 时按 `CWMPID = request_id` 解析命令并持久化出站、入站 XML。
- 增加单元测试覆盖两条修复链路。
