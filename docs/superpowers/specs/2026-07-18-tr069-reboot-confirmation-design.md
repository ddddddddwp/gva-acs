# TR-069 Reboot 执行确认设计

## 背景

当前 GVA 在收到 CPE 的 `RebootResponse` 后立即把 Reboot 命令标记为
`COMPLETED`。这个状态只能证明设备接受了 RPC，不能证明设备实际重启。
BS/OAM 模拟栈已经出现返回 `RebootResponse`、但随后厂商重启逻辑因运行模式
不匹配而未执行的情况。

本变更只修改 GVA TR-069 插件，不修改独立维护的 `tr069-core-only`。

## 目标

- `RebootResponse` 只表示设备已受理命令。
- 设备在受理后上报 `M Reboot` 或 `1 BOOT` Inform，命令才进入
  `COMPLETED`。
- 未在配置时间内收到启动确认的命令进入 `TIMEOUT`。
- 命令记录页面明确区分“等待 RPC 响应”“设备已受理，等待重启”和“完成”。
- 状态和超时均持久化，GVA 重启后仍能继续确认或超时未完成命令。

## 非目标

- 不负责修复 BS 厂商协议栈的 CU/DU 模式重启实现。
- 不把 Docker 容器重启或宿主机重启作为 CWMP Reboot 的完成条件。
- 不修改已经终态化的历史 Reboot 命令。
- 本阶段不修改 core 的 `InformSummary`，因此不依赖 Inform Event 的
  `CommandKey` 做关联。

## 状态机

新增非终态 `WAITING_REBOOT`：

```text
WAITING_DEVICE
    -> BUILDING
    -> SENT
    -> WAITING_REBOOT
    -> COMPLETED
```

异常路径：

```text
SENT -> FAILED
SENT -> TIMEOUT
WAITING_REBOOT -> TIMEOUT
WAITING_REBOOT -> FAILED
```

具体事件：

| 触发条件 | 状态变化 | 事件 | 阶段 |
|---|---|---|---|
| 收到正确关联的 `RebootResponse` | `SENT -> WAITING_REBOOT` | `REBOOT_ACKNOWLEDGED` | `reboot.acknowledged` |
| 后续 Inform 包含 `M Reboot` 或 `1 BOOT` | `WAITING_REBOOT -> COMPLETED` | `REBOOT_CONFIRMED` | `reboot.inform` |
| 确认截止时间到期 | `WAITING_REBOOT -> TIMEOUT` | `REBOOT_CONFIRM_TIMEOUT` | `reboot.confirm` |

`RebootResponse` 到达后，`phase_deadline_at` 设置为当前时间加
`rebootConfirmTimeout`。完成或超时时清空截止时间并写入 `finished_at`。

## 关联策略

GVA 当前从 core 收到的 `InformSummary` 只有 EventCode 列表，没有 Event
CommandKey。首版使用以下约束关联：

1. Inform 已通过 OUI 和 SerialNumber 解析到唯一 GVA 设备。
2. 只查询该设备状态为 `WAITING_REBOOT` 的最早命令。
3. 设备 FIFO 保证同一设备不会同时执行多个非终态命令。
4. 只有在命令进入 `WAITING_REBOOT` 之后收到的启动事件才能确认该命令。

Reboot 命令仍生成并持久化唯一 `CommandKey`，下发 XML 使用该值，为后续
core 暴露 Inform Event CommandKey 后升级成精确关联保留条件。首版不能声称
CommandKey 已参与确认匹配。

EventCode 匹配采用去除首尾空白后的大小写不敏感精确匹配，仅接受：

- `M Reboot`
- `1 BOOT`

普通 `2 PERIODIC`、`4 VALUE CHANGE`、`0 BOOTSTRAP` 不确认 Reboot。

## 后端组件

### Reboot 应答转换

`GormCommandRepo.MarkSuccess` 读取当前命令。普通 RPC 保持原有
`SENT -> COMPLETED`；Reboot 改为 `SENT -> WAITING_REBOOT`，并记录新的截止
时间和事件。Download/Upload 的现有 TransferComplete 逻辑不变。

### Inform 确认器

在 `GormDeviceRepo.UpsertFromInform` 完成设备持久化后调用独立的
`RebootConfirmationService.ConfirmFromInform`。确认器负责：

- 判断 EventCode；
- 按设备锁定最早的 `WAITING_REBOOT` 命令；
- 使用带期望版本的条件状态更新完成命令；
- 重复 Inform 或并发 Inform 不重复完成命令。

确认失败只记录错误，不影响 InformResponse，避免设备因网管内部状态问题
反复重试整个 Inform 会话。

### 截止时间扫描器

新增数据库扫描器，定期把已过 `phase_deadline_at` 的
`WAITING_REBOOT` 命令条件更新为 `TIMEOUT`。扫描使用有限批次和主键顺序，
通过状态与版本条件更新保证多实例幂等。

扫描器随 TR-069 插件启动，并在 GVA shutdown hook 中停止。数据库是事实
来源，因此服务重启后不需要 Redis 恢复等待重启状态。

## 配置

在现有 `tr069` 配置下新增：

```yaml
tr069:
    rebootConfirmTimeout: 300
```

单位为秒，默认 300 秒，必须大于零。配置沿用现有原子运行时快照和热加载
机制；新收到的 `RebootResponse` 使用当时的配置计算固定截止时间，配置变更
不追溯修改已等待命令。

扫描周期属于内部实现常量，默认 5 秒，不暴露额外配置。

## 前端

命令记录状态增加：

```text
WAITING_REBOOT = 设备已受理，等待重启
```

列表的阶段提示显示“等待设备重新上线”。详情时间线展示
`REBOOT_ACKNOWLEDGED`、`REBOOT_CONFIRMED` 或
`REBOOT_CONFIRM_TIMEOUT`。只有 `FAILED` 和 `TIMEOUT` 允许重新下发；
`WAITING_REBOOT` 不允许重复下发。

## 错误处理

- RebootResponse 无法关联命令：保持现有诊断日志，不创建虚假记录。
- Inform 确认数据库失败：记录设备、事件和错误；正常响应 Inform。
- 重复启动 Inform：条件更新无变化，保持已完成状态。
- 超时与启动 Inform 并发：只有一个条件更新成功，最终只能是
  `COMPLETED` 或 `TIMEOUT`。
- GVA 在等待期间重启：状态和截止时间从数据库恢复。

## 测试

### 后端

- Reboot `MarkSuccess` 从 `SENT` 转为 `WAITING_REBOOT`，并设置截止时间。
- 普通 RPC `MarkSuccess` 仍直接完成。
- Reboot 自动生成并在下发参数中携带唯一 CommandKey。
- `M Reboot` 和 `1 BOOT` 可分别确认命令。
- Periodic、Value Change 和 Bootstrap 不确认命令。
- 重复/并发 Inform 不产生重复完成事件。
- 过期等待命令变成 `TIMEOUT`，未过期命令保持不变。
- 确认与超时竞争时只有一个终态。
- 配置默认值、显式值和热加载快照正确。

### 前端

- `WAITING_REBOOT` 映射为“设备已受理，等待重启”。
- 状态过滤器显示“等待设备重新上线”。
- `WAITING_REBOOT` 不显示重试入口。

### 联调

- 下发 Reboot 后首先看到 `WAITING_REBOOT`。
- 能实际重启的 BS 上报 `M Reboot/1 BOOT` 后进入 `COMPLETED`。
- 当前模式无法重启的 BS 在 5 分钟后进入 `TIMEOUT`，不能再误显示为完成。

