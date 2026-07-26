# Brainstorm Summary

- Change: secure-tr069-credential-authentication
- Date: 2026-07-26

## 确认的技术方案

采用TR-069插件内“共享认证核心 + 通道薄适配”架构。四个全局Profile通过五个认证通道映射使用，CWMP接入和Connection Request默认共享Connection Profile。凭据必须成对为空或非空，加密存储并以revision驱动设备状态失效。CWMP认证在正文解析和设备注册之前完成；Connection Request停止自动Provisioning并只使用全局Profile。

## 关键取舍与风险

全局凭据降低配置成本但扩大泄漏影响面；通过主密钥、HTTPS、脱敏和审计缓解。认证失败前没有可信设备身份，因此只记来源IP，不修改正式设备状态。旧自动凭据保留审计但停止使用，用户负责在基站侧人工更新。

## 测试策略

先写模型、原子配置、Basic/Digest协议向量、CWMP阻断和Connection Request无SPV失败测试，再实现；补充API/前端、秘密扫描、回归和两台Docker BS黑盒验收。

## Spec Patch

无。
