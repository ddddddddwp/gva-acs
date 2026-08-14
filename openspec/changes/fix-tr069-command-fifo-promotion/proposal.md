## Why

TR-069 设备的前序 RPC 进入终态后，后续命令仍永久停留在 `QUEUED`，导致能力同步和参数查询/配置无法继续。当前 `8CE468-SNB1234567892` 已稳定复现该问题，需要修复 FIFO 生命周期并恢复现有队列。

## What Changes

- 当前序命令完成、失败或超时时，按创建顺序原子提升下一条 `QUEUED` 命令为 `WAITING_DEVICE`。
- 为被提升的命令设置等待时间和设备等待超时，并写入状态事件。
- 提升成功后写入 Redis 唤醒令牌；Redis 失败不回滚数据库事实，并保留可诊断事件。
- 服务启动时恢复只有 `QUEUED`、没有活动命令的设备队列，并重新唤醒已有 `WAITING_DEVICE` 队首。
- 增加终态推进、启动恢复、并发幂等和 Redis 失败的回归测试。
- 清理 `8CE468-SNB1234567892` 的重复 `GetRPCMethods`，只保留一条继续执行。

## Capabilities

### New Capabilities

- `tr069-command-fifo-recovery`: 保证同设备 RPC 队列在终态后继续推进，并能在服务启动时恢复孤立队列。

### Modified Capabilities

无。本次恢复既有 FIFO 和服务重启恢复设计的预期行为，不改变公开接口或产品能力。

## Impact

- 后端 TR-069 命令状态机、终态处理、启动恢复和 Redis 唤醒。
- 不修改数据库结构，不修改管理 API，不修改前端。
- 运行数据仅处理设备 `8CE468-SNB1234567892` 的重复排队命令。
