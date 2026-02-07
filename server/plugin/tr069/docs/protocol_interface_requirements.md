# 协议层与应用层开发边界及接口需求 (Interface Requirements)

本文档明确界定了 **应用层 (GVA Plugin)** 与 **协议层 (SDK Core)** 的开发边界，并详细列出了应用层对协议层的接口需求。

## 1. 职责边界划分

### 应用层 (Application Layer - GVA Plugin)
*   **位置**: `server/plugin/tr069`
*   **职责**:
    *   **业务逻辑**: 决定“什么时候”下发“什么配置”（如：收到 BOOT 事件 -> 查询数据库 -> 下发配置）。
    *   **数据存储**: 管理设备表、日志表、任务队列（MySQL/Redis）。
    *   **用户界面**: 提供 Web 页面供运维人员操作。
    *   **调度**: 处理异步任务、告警通知。

### 协议层 (Protocol Layer - SDK Core)
*   **位置**: `server/plugin/tr069/lib/tr069-core-only`
*   **职责**:
    *   **报文解析**: 将原始 XML `[]byte` 转换为 Go Struct (`TR069.Message`)。
    *   **报文构建**: 将 Go Struct 序列化为符合 SOAP 标准的 XML。
    *   **数据模型适配**: 处理 TR-098/TR-181/TR-196 等不同数据模型的结构差异。
    *   **安全通道**: 维护 TLS 连接、证书校验、加解密。

---

## 2. 协议层接口需求 (SDK Interface Requirements)

为了支持基站业务开发，协议层需要提供以下具体的 Go 接口和数据结构。

### 2.1 解析器接口 (Parser)

**应用层输入**: `[]byte` (HTTP Request Body)
**协议层输出**: `*tr069.Message` (结构化数据)

**需求细节**:
1.  **支持 TR-196 命名空间**:
    *   必须能识别 `<ParameterValueStruct>` 中的 `Name` 字段包含 `Device.Services.FAPService` 前缀。
    *   **特殊要求**: 对于基站厂商（如华为/中兴）可能的私有参数（如 `X_HW_Power`），解析器不能报错，应作为 `UnknownParameter` 或通用的 `String` 类型返回。
2.  **事件代码解析**:
    *   结构体 `tr069.Event` 必须包含 `EventCode` (字符串) 和 `CommandKey` (字符串)。
    *   例如：能准确解析 `1 BOOT`, `2 PERIODIC`, `M Reboot`。

```go
// 期望的 SDK 接口定义
type Parser interface {
    // 能够容忍未知字段的解析
    ParseMessage(ctx context.Context, rawXML []byte) (*tr069.Message, error)
}
```

### 2.2 构建器接口 (Builder)

**应用层输入**: `*tr069.Message` (包含要下发的参数)
**协议层输出**: `[]byte` (HTTP Response Body)

**需求细节**:
1.  **类型自动推断**:
    *   应用层传入 `interface{}` 类型的 Value，SDK 需根据 TR-069 标准自动转换为 XML 的 `xsi:type` (如 `xsd:unsignedInt`, `xsd:boolean`)。
    *   **重点**: 基站的 PCI 是 `unsignedInt`，频点是 `unsignedInt`，发射功率可能是 `int` 或 `string`。SDK 需要提供手动指定类型的能力。
2.  **大包支持**:
    *   构建 `Download` 或 `Upload` 报文时，如果包含其它 Payload，确保 SOAP Body 格式正确。

```go
// 期望的 SDK 接口定义
type Builder interface {
    // 构建 SetParameterValues 请求
    BuildSetParameterValues(params map[string]interface{}) ([]byte, error)
    
    // 构建带有明确类型的参数请求 (用于解决类型模糊问题)
    BuildTypedSetParameterValues(params []tr069.ParameterValue) ([]byte, error)
}
```

### 2.3 连接安全配置 (Security)

**需求细节**:
1.  **mTLS 配置入口**:
    *   SDK 需要提供一个配置对象，允许应用层传入证书路径或证书内容。
    *   底层 `http.Server` 启动时，自动加载这些证书。

```go
// 期望的配置结构
type SecurityConfig struct {
    CACert     []byte // 根证书，用于校验基站
    ServerCert []byte // 服务端公钥
    ServerKey  []byte // 服务端私钥
    RequireClientCert bool // 是否强制验证基站证书
}
```

---

## 3. 开发协作流程

1.  **应用层开发 (我负责)**:
    *   我会在 `handler` 中调用 SDK 的 `ParseMessage`。
    *   如果发现解析出来的结构体中，某些 TR-196 的关键参数（如 `PhysicalCellID`）丢失或乱码，我会记录 **[SDK Issue]**。
    *   我会构造业务数据，调用 SDK 的 `BuildMessage`。如果基站返回 `9003 Invalid Arguments`，说明 SDK 构建的 XML 类型有误，我会反馈。

2.  **协议层开发 (您协调)**:
    *   您需要确保 SDK 能够正确处理上述输入输出。
    *   **交付物**: 更新后的 `tr069-core-only` 仓库 (V2 分支)。

---

## 4. 立即行动计划 (Action Items)

### 步骤 1: 确认 TR-196 解析能力
*   **动作**: 我将提供一个标准的基站 `Inform` 报文样例（包含 `FAPService` 参数）。
*   **验收**: 请您协调协议层，确保 SDK 的单元测试能跑通这个样例，并正确提取出 `PhysicalCellID`。

### 步骤 2: 确认 mTLS 接口
*   **动作**: 我将在 `config.yaml` 中增加证书路径配置。
*   **验收**: 请协议层提供 `NewServerTLSConfig(cert, key, ca)` 这样的辅助函数，供应用层启动 HTTP 服务时使用。
