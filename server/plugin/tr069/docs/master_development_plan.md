# TR-069 基站/专网设备管理系统开发计划 (Master Plan)

本文档基于 GVA 架构与 TR-069 协议栈，结合基站（Small Cell/Femtocell）与专网设备的业务特性，制定了全方位的开发计划。本计划涵盖了从底层协议适配到上层业务逻辑的完整开发周期。

---

## 阶段一：核心架构与连接层 (Foundation Phase)

**目标**: 建立高可靠、安全的 ACS 与基站通信链路，实现设备接入与基础管理。

### 1.1 SDK 协议栈强化 (SDK Enhancement)
*   **TR-196 数据模型适配**:
    *   **任务**: 扩展 XML 解析器，支持 `Device.Services.FAPService.{i}.` (Femto Access Point) 命名空间。
    *   **SDK Action**: 处理 Vendor-Specific 参数（如 `X_VENDOR_Specific`），确保解析器具备容错性，不因未知字段而崩溃。
*   **安全认证模块升级**:
    *   **任务**: 实现双向 TLS (mTLS) 认证。
    *   **SDK Action**: 在 `net/http` Server 中集成 TLS Config，支持加载 CA 根证书、服务端证书及私钥，并强制校验客户端证书。
*   **XML/SOAP 协议优化**:
    *   **SDK Action**: 优化 SOAP Envelope 的封装与解包效率，支持大包传输（如 Upload/Download 场景）。

### 1.2 会话与连接管理 (Session Management)
*   **Inform 消息状态机**:
    *   **任务**: 开发 Inform 处理逻辑，正确响应 `1 BOOT` (启动), `2 PERIODIC` (周期), `4 VALUE CHANGE` (变参), `6 CONNECTION REQUEST` (连接请求) 等核心事件。
    *   **技术**: 使用 Redis 缓存 Session 上下文（DeviceID <-> SessionID），支持高并发连接。
*   **心跳保活与超时处理**:
    *   **任务**: 针对不稳定的回传网络（Backhaul），优化心跳检测逻辑，处理 "Empty Post" 以优雅关闭会话。

### 1.3 设备接入与权限控制 (Device Onboarding)
*   **白名单机制**:
    *   **任务**: 建立 `base_station_whitelist` 表，仅允许录入 SN/OUI 的设备接入。
*   **设备信息档案**:
    *   **任务**: 在设备首次上线 (`BOOT`) 时，自动采集并存储 `Device.DeviceInfo` (Model, SoftwareVersion, HardwareVersion) 及网络信息 (IP, MAC)。

---

## 阶段二：无线参数与状态监控 (Radio & Monitoring Phase)

**目标**: 实现对基站无线射频参数的配置下发，以及运行状态的实时监控。

### 2.1 射频参数管理 (RF Configuration)
*   **小区参数配置**:
    *   **任务**: 实现对 `FAPService.{i}.CellConfig` 下关键参数的读写：
        *   `PhysicalCellID` (PCI): 物理小区标识。
        *   `EARFCN` (4G) / `NR-ARFCN` (5G): 频点信息。
        *   `TxPower`: 发射功率控制。
*   **邻区关系管理 (Neighbor Relations)**:
    *   **任务**: 开发邻区列表 (`NeighborList`) 的增删改查接口，支持配置自动邻区关系（ANR）开关。
    *   **SDK Action**: 封装 `AddObject` / `DeleteObject` 的便捷调用接口，用于动态管理邻区列表。

### 2.2 实时状态与性能监控 (KPI Monitoring)
*   **KPI 采集引擎**:
    *   **任务**: 利用 `PeriodicInform` 机制，周期性采集关键指标：
        *   **用户面**: RRC 连接数 (`RRC.Connected`), 吞吐量 (`Throughput`).
        *   **同步面**: GPS/北斗锁定状态 (`GPS.Locked`), PTP 同步状态.
    *   **数据流**: Inform -> 解析 -> Redis (实时快照) -> MySQL/TimescaleDB (历史趋势).
*   **告警处理 (Fault Management)**:
    *   **任务**: 监听基站上报的 `Alarm` 事件 (如驻波比异常、RRU 过热、S1 接口中断)，触发系统告警。

---

## 阶段三：高级运维与自动化 (Advanced Operations Phase)

**目标**: 降低运维成本，实现批量管理、故障诊断与自愈。

### 3.1 批量操作与自动化 (Bulk Operations)
*   **批量固件升级**:
    *   **任务**: 开发任务调度系统，支持按区域、型号筛选基站，批量下发 `Download` 指令进行升级。
*   **配置快照与回滚**:
    *   **任务**: 每次关键配置变更前，自动触发 `Upload` 备份当前配置文件，支持一键回滚至历史版本。

### 3.2 远程诊断工具箱 (Diagnostics Toolkit)
*   **网络诊断**:
    *   **任务**: Web 端发起 Ping/TraceRoute 指令，基站执行后异步上报结果，用于排查回传网络故障。
*   **射频环境扫描 (REM)**:
    *   **任务**: 下发 REM 扫描指令，获取周边基站的信号强度与干扰情况，辅助网优。

### 3.3 商业化运营支撑 (Business Support)
*   **库存与生命周期**:
    *   **任务**: 记录设备入库、安装、维修、报废全流程，关联装维人员信息。
*   **IPSec 隧道监控**:
    *   **任务**: 监控设备与核心网之间的 IPSec 隧道状态，确保业务数据传输安全。

---

## 开发协作与 SDK 支持约定

在开发过程中，我们将遵循以下协作模式：

1.  **业务层 (Business Layer)**: 您主要负责 GVA 框架内的 API 接口、数据库模型 (Model)、Service 业务逻辑以及前端页面开发。
2.  **协议层 (Protocol Layer)**: 凡涉及底层 XML 协议解析、SOAP 报文封装、TLS 握手细节、TR-196 模型兼容性等问题，请标记为 **[SDK Task]**。
    *   **反馈方式**: "SDK 无法解析这个 XML 结构" 或 "需要支持新的 Vendor RPC 方法"。
    *   **响应机制**: 我将直接修改 `server/plugin/tr069/lib/tr069-core-only` 下的源码进行修复或扩展，并同步更新到本地环境中。
