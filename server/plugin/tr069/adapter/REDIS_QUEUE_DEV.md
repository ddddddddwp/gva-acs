# TR069 Redis 队列设计

## 队列架构

### 数据结构

```
┌─────────────────────────────────────────────────────────────┐
│  Redis List (按设备分开)                                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  immediate 队列 (主动下发):                                 │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ tr069:cmd:immediate:{deviceKey}                      │  │
│  │   [cmd1] → [cmd2] → [cmd3]                          │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  pending 队列 (被动下发):                                   │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ tr069:cmd:pending:{deviceKey}                       │  │
│  │   [cmd1] → [cmd2] → [cmd3] → [cmd4] → [cmd5] ...  │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 队列说明

| 队列 | 用途 | 写入方式 | 消费者 |
|------|------|----------|--------|
| `immediate` | 主动下发（高优先级） | `enqueueImmediate()` | 设备立即拉取 |
| `pending` | 被动下发（普通） | Dispatcher 从 Stream 消费后写入 | 设备下次 Inform 时拉取 |

### Pull 流程（优先级处理）

```
设备拉取命令
     │
     ▼
1. 检查 immediate 队列（主动下发）
   │ 有命令 → 立即返回，优先处理
   │ 无命令 → 继续
   │
2. 检查 pending 队列（被动下发）
   │ 最多处理 N 条（可配置，避免饥饿）
   │ 超过则等待下次 session
   │
3. 释放设备锁
```

---

## Redis Key 定义

| Key 前缀 | 说明 | 示例 |
|----------|------|------|
| `tr069:cmd:ingest` | Redis Stream | 全局命令收集 |
| `tr069:cmd:dispatchers` | 消费者组 | Stream 消费者组 |
| `tr069:cmd:immediate:{deviceKey}` | 主动下发队列 | 按设备隔离 |
| `tr069:cmd:pending:{deviceKey}` | 被动下发队列 | 按设备隔离 |
| `tr069:cmd:lock:{deviceKey}` | 设备分布式锁 | 确保单消费者 |
| `tr069:cmd:dedup:{dedupKey}` | 去重 Key | 防止重复执行 |

---

## 消费稳定性保障

### 1. 设备级别分布式锁

```go
// 确保同一时间只有一个消费者处理同一设备
lockKey := "tr069:cmd:lock:" + deviceKey
ok, err := client.SetNX(ctx, lockKey, instanceID, 30*time.Second).Result()
```

### 2. 去重机制 (Dedup)

```go
// 使用 SETNX 实现去重，防止同一命令重复执行
dedupKey := "tr069:cmd:dedup:" + dedupKey
seen, err := client.SetNX(ctx, dedupKey, "1", 24*time.Hour).Result()

if err == nil && !seen {
    // key已存在，说明命令刚执行过，跳过
    continue
}
```

### 3. Pending 数量限制

每次 session 最多处理一定数量的 pending 命令，避免：
- 设备长时间占用
- 被动命令"饥饿"（永远处理不到）

---

## 配置文件

### config.yaml 配置项

```yaml
tr069:
    # ==================== 队列配置 ====================
    
    # commandQueueLockTTL: 设备级别分布式锁的 TTL（秒）
    # 说明: 获取设备命令时的锁超时时间
    # 默认: 30
    commandQueueLockTTL: 30
    
    # commandQueueDedupTTL: 去重 key 的 TTL（秒）
    # 说明: 同一命令在指定时间内不重复执行
    # 默认: 86400 (24小时)
    commandQueueDedupTTL: 86400
    
    # commandQueueMaxScan: 每次从队列获取命令的最大扫描次数
    # 说明: 遍历队列寻找可用命令的最大次数
    # 默认: 10
    commandQueueMaxScan: 10
    
    # commandQueueMaxPendingPerSession: 每次 session 最多处理的 pending 命令数量
    # 说明: 避免被动命令饥饿，设备每次连接最多处理数量
    # 默认: 5
    commandQueueMaxPendingPerSession: 5
    
    # commandQueueImmediateTTL: 立即下发队列的 TTL（秒）
    # 说明: immediate 队列中命令的过期时间
    # 默认: 1800 (30分钟)
    commandQueueImmediateTTL: 1800
```

### 配置项说明

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `commandQueueLockTTL` | 30秒 | 设备级别分布式锁的 TTL |
| `commandQueueDedupTTL` | 86400秒 (24小时) | 去重 key 的 TTL，防止重复执行 |
| `commandQueueMaxScan` | 10 | 每次从队列获取命令的最大扫描次数 |
| `commandQueueMaxPendingPerSession` | 5 | 每次 session 最多处理的 pending 命令数量 |
| `commandQueueImmediateTTL` | 1800秒 (30分钟) | 立即下发队列的 TTL |

---

## 相关文件

| 文件 | 说明 |
|------|------|
| `adapter/redis_command_source.go` | Redis 队列消费者，Pull 实现 |
| `adapter/redis_keys.go` | Redis Key 常量定义 |
| `adapter/redis_immediate_enqueue.go` | 主动下发实现 |
| `adapter/redis_dispatcher.go` | Stream 消费者 |
| `config/config.go` | 配置结构体定义 |
| `global/global.go` | 全局配置变量 |
| `initialize/viper.go` | 配置加载逻辑 |
