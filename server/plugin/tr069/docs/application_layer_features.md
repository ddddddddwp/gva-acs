# TR-069 应用层功能开发指南

本文档列举了基于 TR-069 协议进行应用层（ACS - Auto Configuration Server）开发时，常见的核心功能模块与业务场景。这些功能构成了 CPE（Customer Premises Equipment）管理系统的基础。

## 1. 基础连接与会话管理 (Session Management)

这是 TR-069 的基石，负责 CPE 与 ACS 之间的通信建立与维护。

*   **Inform 消息处理**:
    *   **功能**: 接收并解析 CPE 发来的 `Inform` 消息（周期性上报、启动上报、值变更上报等）。
    *   **关键点**: 验证 CPE 身份（SN, OUI, ProductClass），解析 Event Code（如 `1 BOOT`, `2 PERIODIC`），回复 `InformResponse`。
*   **连接认证 (Authentication)**:
    *   **功能**: 实现 HTTP Basic/Digest 认证，确保只有合法的 CPE 能接入。
    *   **关键点**: 验证用户名密码，处理 SSL/TLS 证书（如果启用 HTTPS）。
*   **连接请求 (Connection Request)**:
    *   **功能**: ACS 主动发起连接请求（Connection Request），“踢” CPE 上线。
    *   **关键点**: 保存 CPE 的 ConnectionRequestURL，处理 NAT 穿透问题（STUN/UDP Connection Request）。
*   **空闲超时与会话结束**:
    *   **功能**: 判定会话结束（CPE 发送空包），释放服务器资源。

## 2. 设备管理 (Device Management)

针对单台设备的生命周期管理。

*   **设备信息采集 (GetParameterValues)**:
    *   **功能**: 获取设备的基础信息和状态。
    *   **常用参数**:
        *   `Device.DeviceInfo.HardwareVersion`
        *   `Device.DeviceInfo.SoftwareVersion`
        *   `Device.DeviceInfo.UpTime`
        *   `Device.WANConnectionDevice.{i}.ExternalIPAddress`
*   **设备参数配置 (SetParameterValues)**:
    *   **功能**: 下发配置修改设备的运行参数。
    *   **场景**: 修改 Wi-Fi SSID/密码、修改 WAN 口拨号账号、设置 NTP 服务器等。
*   **设备重启 (Reboot)**:
    *   **功能**: 远程控制设备重启。
*   **恢复出厂设置 (FactoryReset)**:
    *   **功能**: 远程将设备恢复到出厂默认状态。
*   **对象增删 (AddObject/DeleteObject)**:
    *   **功能**: 动态添加或删除多实例对象。
    *   **场景**: 添加一个新的 WAN 连接配置，删除一个端口映射规则。

## 3. 固件与文件管理 (Firmware & File Management)

*   **固件升级 (Download)**:
    *   **功能**: 指示 CPE 下载并刷写新的固件镜像。
    *   **关键点**: 支持文件服务器（HTTP/FTP），支持延迟升级（Schedule），上报升级结果（TransferComplete）。
*   **日志/配置上传 (Upload)**:
    *   **功能**: 指示 CPE 上传日志文件或当前配置文件到服务器。
    *   **场景**: 故障诊断时获取设备系统日志。

## 4. 监控与诊断 (Monitoring & Diagnostics)

*   **状态监控**:
    *   **功能**: 实时或周期性监控关键指标（CPU, 内存, 网络流量, 光功率）。
*   **故障诊断 (Diagnostics)**:
    *   **功能**: 触发设备执行内置的诊断工具。
    *   **常用诊断**:
        *   `Ping` 测试（连通性检测）
        *   `TraceRoute`（路由追踪）
        *   `SpeedTest`（测速，部分扩展模型支持）
*   **告警管理**:
    *   **功能**: 监听特定的值变更事件（ValueChange）。
    *   **场景**: 当 WAN 口 IP 变更、光信号丢失或设备重启时，ACS 生成告警记录。

## 5. 业务发放 (Service Provisioning)

针对特定业务场景的自动化配置流程。

*   **零配置上线 (Zero Touch Provisioning)**:
    *   **功能**: 设备开箱上电后，自动连接 ACS 并下载所有基础配置，无需人工干预。
*   **VoIP 语音配置**:
    *   **功能**: 下发 SIP 服务器地址、账号、密码、数图等配置。
*   **IPTV 配置**:
    *   **功能**: 下发 IPTV VLAN、组播 VLAN 等配置。

## 6. 扩展功能 (Extensions)

*   **脚本化/工作流引擎**:
    *   **功能**: 定义一系列操作步骤（如：先查版本 -> 这里的版本低 -> 下发升级 -> 等待重启 -> 下发配置），自动化执行复杂任务。
*   **批量操作 (Batch Operations)**:
    *   **功能**: 对满足特定条件（如同一型号、同一区域）的一批设备执行相同的操作（如批量升级）。

## 7. 商业化运营支撑 (Business Support)

*   **库存与生命周期管理 (Lifecycle Management)**:
    *   **功能**: 记录设备从入库、出库、安装、维修到报废的全过程。
    *   **场景**: 关联工单系统，追踪哪个安装师傅安装了哪台设备。
*   **套餐与QoS绑定 (SLA Enforcement)**:
    *   **功能**: 根据用户的签约套餐（如 100M/500M），动态下发 QoS 限速模板。
    *   **价值**: 确保高价值用户体验，防止带宽滥用。
*   ** captive portal 集成**:
    *   **功能**: 欠费用户重定向到缴费页面。
    *   **实现**: 下发路由策略，将 80 端口流量劫持到运营商门户。

## 8. 高级排障工具 (Advanced Troubleshooting)

*   **Wi-Fi 邻居扫描与优化**:
    *   **功能**: 让 CPE 扫描周围 Wi-Fi 信号强度和信道占用情况。
    *   **价值**: ACS 自动计算并下发最优信道，解决 Wi-Fi 干扰导致的慢速问题。
*   **回路检测 (Loop Detection)**:
    *   **功能**: 监测 LAN 侧是否存在环路。
    *   **价值**: 防止用户误接网线导致全家断网。

---

**技术实现建议 (基于 GVA + Redis)**:

1.  **缓存层**: 使用 Redis 缓存设备的 Session 状态、待下发的任务队列（Job Queue），提高并发处理能力。
2.  **异步处理**: 对于耗时的操作（如升级、诊断），ACS 应立即响应 HTTP 请求，通过任务队列异步追踪 CPE 的 `TransferComplete` 或后续 `Inform` 消息来更新结果。
3.  **数据分层**: 
    *   **热数据**: Session、实时状态 -> Redis
    *   **业务数据**: 设备列表、配置模板 -> MySQL
    *   **历史数据**: 性能监控日志、操作审计 -> 建议引入时序数据库或日志系统 (Elasticsearch/ClickHouse)
