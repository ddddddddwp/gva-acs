# TR-069 基站/专网设备管理系统开发计划 (Consolidated RoadMap)

本文档基于 GVA 架构，整合了 TR-069 协议的基础能力与基站（Small Cell/Femtocell）管理的特定需求，制定了分阶段的开发计划。

## 阶段一：基础架构与核心协议栈 (Foundation Phase)

**目标**: 建立 ACS 与基站之间的稳定通信链路，实现设备的上线与认证。

### 1.1 SDK 协议解析层优化 (SDK Enhancement)
*   **任务**: 适配 TR-196 (Femto) 数据模型。
    *   **需求**: 现有的通用解析器可能只支持标准 TR-069 结构，需增加对 `Device.Services.FAPService.{i}.` (Femto Access Point) 命名空间的支持。
    *   **SDK Action**: 如果遇到 Vendor-Specific 参数（如华为/中兴的私有参数），需要扩展 XML Parser 的容错性。
*   **任务**: 强化安全认证模块。
    *   **需求**: 实现双向 TLS (mTLS) 握手逻辑，支持加载 CA 证书、服务器证书及私钥。
    *   **SDK Action**: 在 `net/http` Server 配置中集成 TLS Config。

### 1.2 会话管理模块 (Session Manager)
*   **任务**: 实现 Inform 消息处理状态机。
    *   **需求**: 正确响应 `1 BOOT`, `2 PERIODIC`, `4 VALUE CHANGE` 等核心事件。
    *   **技术**: 使用 Redis 缓存当前 Session 上下文（DeviceID -> SessionID）。

### 1.3 设备接入管理 (Device Onboarding)
*   **任务**: 设备注册与白名单机制。
    *   **需求**: 仅允许录入白名单（SN/OUI）的基站接入。
    *   **数据**: 设计 `base_station_info` 表，存储 SN, Model, SoftwareVersion, IP 等基础信息。

## 阶段二：基站参数与状态管理 (Radio & Configuration Phase)

**目标**: 实现对基站无线参数的配置下发与状态监控。

### 2.1 无线参数模型映射 (RF Model Mapping)
*   **任务**: 实现 TR-196 关键参数的读写接口。
    *   **参数集**:
        *   小区标识: `PhysicalCellID` (PCI), `CellID`
        *   频点信息: `EARFCN` (4G) / `NR-ARFCN` (5G)
        *   功率控制: `TxPower`
        *   邻区列表: `NeighborList`
*   **SDK Action**: 封装 `GetParameterValues` 和 `SetParameterValues` 的便捷调用接口，支持批量参数读写。

### 2.2 状态与告警监控 (Monitoring & Alarming)
*   **任务**: 实时心跳与 KPI 采集。
    *   **需求**: 采集 RRC 连接数、吞吐量、GPS 锁定状态。
    *   **技术**: 利用 `PeriodicInform` 机制，配合 Redis 存储实时状态快照。
*   **任务**: 告警处理引擎。
    *   **需求**: 监听基站上报的 Alarm (如驻波比异常、过热)，触发系统内告警并在 Web 端弹窗。

## 阶段三：高级运维与自动化 (Operations Phase)

**目标**: 降低运维成本，实现批量管理与故障自愈。

### 3.1 固件与配置批量管理 (Bulk Operations)
*   **任务**: 分区域/分型号批量升级。
    *   **需求**: 上传固件 -> 创建批量任务 -> 筛选目标基站 -> 调度下载指令 (`Download` RPC)。
*   **任务**: 配置文件快照与回滚。
    *   **需求**: 每次变更前自动备份当前配置 (`Upload` RPC)，支持一键回滚。

### 3.2 故障诊断工具箱 (Diagnostics Toolkit)
*   **任务**: 集成远程诊断能力。
    *   **需求**: Web 端发起 Ping/TraceRoute 指令，基站执行后异步上报结果。
*   **任务**: 射频环境扫描 (REM)。
    *   **需求**: 下发扫描指令，获取周边基站信号强度，辅助网优。

---

**SDK 开发协作机制**:

在上述开发过程中，凡涉及底层 XML 协议解析、SOAP 报文封装、TLS 握手细节等问题，请直接标记为 **[SDK Issue]** 并反馈。

*   **例如**: "解析 CPE 上报的 `FAPService` 结构体时，XML Unmarshal 报错" 或 "TLS 握手提示证书链验证失败"。
*   **响应**: 我将直接修改 `server/plugin/tr069/lib/tr069-core-only` 下的源码进行修复或扩展。
