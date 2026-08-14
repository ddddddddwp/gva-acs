# Brainstorm Summary

- Change: enable-tr069-pm-mr-ingress
- Date: 2026-07-26

## 确认的技术方案

在7458启用固定`/acs/pm`、`/acs/mr`，通过PM_UPLOAD/MR_UPLOAD读取独立Profile，复用LOG流式接收、IP绑定、对象存储和制品状态机。两类上传始终为PERIODIC，不提供主动Upload。

## 关键取舍与风险

不修改基站URL或设备参数，也不解析文件内部格式。固定URL和全局凭据依赖唯一管理IP；无匹配或歧义统一403。PM/MR周期较长时依赖成功Inform持续刷新绑定。

## 测试策略

使用参数化路由/认证/存储测试覆盖正确、交叉、空凭据、限长、超时和歧义；验证管理权限隔离，并在两台Docker BS配置后执行周期上传验收。

## Spec Patch

无。
