## ADDED Requirements

### Requirement: 终态自动推进同设备 FIFO
系统 SHALL 在某设备命令进入 `COMPLETED`、`FAILED` 或 `TIMEOUT` 后，原子选择该设备按创建时间最早的 `QUEUED` 命令并将其转换为 `WAITING_DEVICE`，同时设置等待时间和设备等待截止时间。

#### Scenario: 前一条命令完成后执行下一条
- **WHEN** 同一设备的活动头命令进入 `COMPLETED` 且仍有多条 `QUEUED` 命令
- **THEN** 系统只将最早的一条 `QUEUED` 命令转换为 `WAITING_DEVICE`
- **AND** 其余命令保持 `QUEUED`

#### Scenario: 前一条命令失败或超时后执行下一条
- **WHEN** 同一设备的活动头命令进入 `FAILED` 或 `TIMEOUT`
- **THEN** 系统以相同 FIFO 规则推进下一条命令

#### Scenario: 活动命令阻止越序推进
- **WHEN** 设备仍存在 `WAITING_DEVICE`、`BUILDING`、`SENT` 或 `WAITING_TRANSFER` 命令
- **THEN** 系统 MUST NOT 提升任何 `QUEUED` 命令

### Requirement: 推进后触发设备唤醒
系统 SHALL 在 `QUEUED -> WAITING_DEVICE` 数据库事务提交后写入 Redis 唤醒令牌，从而复用现有 Connection Request 流程。

#### Scenario: Redis 入队成功
- **WHEN** 下一条命令成功进入 `WAITING_DEVICE`
- **THEN** 系统写入该设备的 Redis 唤醒令牌

#### Scenario: Redis 入队失败
- **WHEN** 命令已经进入 `WAITING_DEVICE` 但 Redis 入队失败
- **THEN** 系统保留数据库状态并记录可诊断事件
- **AND** 系统 MUST NOT 将已推进命令回滚为 `QUEUED`

### Requirement: 服务启动恢复异常队列
系统 SHALL 在服务启动时恢复只有 `QUEUED`、没有活动头命令的设备队列，并为已有 `WAITING_DEVICE` 队首重新写入唤醒令牌。

#### Scenario: 恢复孤立排队命令
- **WHEN** 服务启动时设备只有一个或多个 `QUEUED` 命令且不存在活动命令
- **THEN** 系统按 FIFO 规则提升最早命令并触发唤醒

#### Scenario: 恢复已有等待命令
- **WHEN** 服务启动时设备队首已经是 `WAITING_DEVICE`
- **THEN** 系统保持命令状态并重新写入唤醒令牌

#### Scenario: 重复恢复保持幂等
- **WHEN** 启动恢复或终态推进被重复或并发调用
- **THEN** 每台设备仍最多只有一个活动头命令
