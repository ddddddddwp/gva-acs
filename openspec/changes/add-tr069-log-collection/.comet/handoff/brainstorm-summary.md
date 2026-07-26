# Brainstorm Summary

- Change: add-tr069-log-collection
- Date: 2026-07-26

## 确认的技术方案

保留现有`/acs/log`、Inform/IP唯一绑定、流式MinIO、ACTIVE/PERIODIC任务和管理页面。认证改由全局LOG Profile提供，空Profile表示无认证。主动Upload RPC仍下发固定URL，但Username/Password永远为空，BS使用本地`Device.LogMgmt.*`凭据完成HTTP上传。

## 关键取舍与风险

空RPC凭据后回退本地凭据是当前BS厂商契约而非通用TR-069保证，必须以两台Docker BS黑盒测试固定。固定URL与全局凭据无法区分同一NAT后的设备，因此沿用唯一IP匹配和歧义403。

## 测试策略

保留已完成的流式、状态机和存储测试；新增空Profile、半配置、主动RPC空凭据、无秘密持久化、两台BS主动/周期上传和revision状态测试，并完成原change遗留API/前端/race任务。

## Spec Patch

已更新`tr069-file-ingress`，加入空LOG Profile和主动Upload空凭据场景。
