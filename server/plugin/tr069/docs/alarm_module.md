# TR-069 告警模块开发文档

## 一、模块概述

TR-069 告警模块负责接收、解析、存储和查询设备上报的告警信息。采用**两表策略**：

| 表名 | 用途 | 数据来源 |
|------|------|----------|
| `tr069_alarms` | 主告警表 | CurrentAlarm + ExpeditedEvent + HistoryEvent |
| `support_tr069_alarms` | 设备支持的告警类型 | SupportedAlarm |

---

## 二、告警协议字段

### TR-069 告警标准字段（共12个）

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `AlarmIdentifier` | string | 告警唯一标识 | `02026021121050568800` |
| `NotificationType` | string | 通知类型 | `NewAlarm` / `ClearedAlarm` |
| `EventTime` | dateTime | 事件时间 | `2026-02-11T13:05:05Z` |
| `EventType` | string | 事件类型 | `Communications Alarm` |
| `ManagedObjectInstance` | string | 管理对象实例 | `DN='DC=BJ,SubNetwork=1,...'` |
| `OUI` | string | 组织唯一标识 | `8CE468` |
| `SerialNumber` | string | 设备序列号 | `SNB123456789` |
| `PerceivedSeverity` | string | 严重程度 | `Critical` / `Major` / `Minor` / `Warning` |
| `ProbableCause` | string | 可能原因 | `Transmission Error` |
| `SpecificProblem` | string | 具体问题 | `EU TO BBU UDP LINK FAIL` |
| `AdditionalText` | string | 附加文本 | `47` |
| `AdditionalInformation` | string | 附加信息 | `EU[1] deviceId=1 raise...` |

### 告警来源类型（Source）

| 来源 | NotificationType | 状态 | 说明 |
|------|------------------|------|------|
| `CurrentAlarm` | 任意 | Active | 当前活跃告警 |
| `ExpeditedEvent` | `NewAlarm` | Active | 新紧急事件 |
| `ExpeditedEvent` | `ClearedAlarm` | Cleared | 已清除紧急事件 |
| `HistoryEvent` | 任意 | Cleared | 历史事件（已清除） |
| `QueuedEvent` | 任意 | - | **忽略**（数据为空） |

### 通知类型（NotificationType）

| 类型 | 说明 | 处理方式 |
|------|------|----------|
| `NewAlarm` | 新告警 | 插入/更新数据库，状态为 Active |
| `ClearedAlarm` | 清除告警 | 更新状态为 Cleared，设置 EndTime |
| `ChangedAlarm` | 变更告警 | 更新数据库 |

---

## 三、数据流架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              设备上报                                    │
│  Device.FaultMgmt.CurrentAlarm.XXX.AlarmIdentifier = "xxx"              │
│  Device.FaultMgmt.ExpeditedEvent.XXX.NotificationType = "NewAlarm"      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          tr069-core 解析层                               │
│  位置: /root/demo/tr069-core/pkg/core/machine.go                        │
│                                                                          │
│  1. handleInform() - 解析 Inform 中的告警参数                           │
│  2. handleResponse() - 解析 GetParameterValuesResponse 中的告警参数     │
│  3. parseAlarms() - 将扁平参数分组为 Alarm 结构体                       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          GVA 适配层                                      │
│  位置: server/plugin/tr069/adapter/gorm_repo.go                         │
│                                                                          │
│  SyncAlarms(ctx, deviceID, alarms)                                      │
│    ├── 过滤 QueuedEvent                                                 │
│    ├── 分离 ClearedAlarm                                                │
│    ├── 批量 Upsert Active 告警                                          │
│    └── 批量更新 Cleared 告警状态                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          数据库存储                                      │
│  表: tr069_alarms                                                        │
│                                                                          │
│  - alarm_identifier: 唯一索引，用于去重和更新                            │
│  - status: Active / Cleared                                             │
│  - source: CurrentAlarm / ExpeditedEvent / HistoryEvent                 │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 四、核心代码解析

### 4.1 tr069-core 解析逻辑

**文件**: `/root/demo/tr069-core/pkg/core/machine.go`

```go
// 告警参数筛选
for _, p := range msg.Parameters {
    if len(p.Name) >= len("Device.FaultMgmt.") && 
       p.Name[:len("Device.FaultMgmt.")] == "Device.FaultMgmt." {
        faults[p.Name] = fmt.Sprint(p.Value)
    }
}

// 解析告警
alarms := parseAlarms(faults)

// 同步到存储
if len(alarms) > 0 && session.DeviceID != "" {
    _ = m.conf.DeviceRepo.SyncAlarms(ctx, session.DeviceID, alarms)
}
```

### 4.2 parseAlarms 分组逻辑

```go
// 参数分组示例
// 输入:
//   Device.FaultMgmt.CurrentAlarm.175.AlarmIdentifier = "xxx"
//   Device.FaultMgmt.CurrentAlarm.175.PerceivedSeverity = "Critical"
//   Device.FaultMgmt.CurrentAlarm.176.AlarmIdentifier = "yyy"

// 输出:
//   group["Device.FaultMgmt.CurrentAlarm.175"] = {AlarmIdentifier: "xxx", PerceivedSeverity: "Critical"}
//   group["Device.FaultMgmt.CurrentAlarm.176"] = {AlarmIdentifier: "yyy"}
```

### 4.3 GVA 批量写入

**文件**: `server/plugin/tr069/adapter/gorm_repo.go`

```go
func (r *GormDeviceRepo) SyncAlarms(ctx context.Context, deviceID string, alarms []core.Alarm) error {
    var toInsert []model.Tr069Alarm
    var toClear []string

    for _, a := range alarms {
        if a.Source == "QueuedEvent" {
            continue  // 忽略队列事件
        }

        var status string
        var endTime *time.Time
        now := time.Now()

        switch a.Source {
        case "CurrentAlarm":
            status = "Active"
        case "ExpeditedEvent":
            if a.NotificationType == "ClearedAlarm" {
                status = "Cleared"
                endTime = &now
                toClear = append(toClear, a.AlarmIdentifier)
            } else {
                status = "Active"
            }
        case "HistoryEvent":
            status = "Cleared"
            endTime = &now
        default:
            status = "Active"
        }
        
        toInsert = append(toInsert, model.Tr069Alarm{
            Status:  status,
            EndTime: endTime,
            // ... 其他字段
        })
    }

    // 批量 Upsert（100条/批）
    r.batchUpsertAlarms(ctx, toInsert)
    
    // 批量清除（更新已存在的活跃告警）
    db.Model(&Tr069Alarm{}).
        Where("alarm_identifier IN ?", toClear).
        Where("status = ?", "Active").
        Updates(map[string]interface{}{
            "status": "Cleared",
            "end_time": time.Now(),
        })
}
```

---

## 五、API 接口

### 5.1 活跃告警列表（默认）

```
GET /api/tr069/alarm/list

Query Parameters:
  - page: 页码（默认1）
  - pageSize: 每页数量（默认10，最大100）
  - serialNumber: 设备序列号（模糊匹配）
  - source: 来源类型
  - severity: 严重程度

Response:
{
  "code": 0,
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "pageSize": 10
  }
}
```

### 5.2 历史告警列表

```
GET /api/tr069/alarm/history

Query Parameters: 同上

Response: 同上
```

### 5.3 告警统计

```
GET /api/tr069/alarm/stats

Response:
{
  "code": 0,
  "data": {
    "total": 1000,
    "active": 50,
    "cleared": 950,
    "critical": 10,
    "major": 20,
    "minor": 15,
    "warning": 5
  }
}
```

---

## 六、数据库表结构

### tr069_alarms

```sql
CREATE TABLE `tr069_alarms` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `device_id` bigint unsigned DEFAULT NULL COMMENT '关联设备ID',
  `serial_number` varchar(64) DEFAULT NULL COMMENT '设备序列号',
  `oui` varchar(64) DEFAULT NULL COMMENT '组织唯一标识',
  `alarm_identifier` varchar(128) DEFAULT NULL COMMENT '告警唯一标识',
  `source` varchar(32) DEFAULT NULL COMMENT '来源',
  `notification_type` varchar(32) DEFAULT NULL COMMENT '通知类型',
  `status` varchar(16) DEFAULT 'Active' COMMENT '状态',
  `event_type` varchar(64) DEFAULT NULL COMMENT '事件类型',
  `perceived_severity` varchar(32) DEFAULT NULL COMMENT '严重程度',
  `probable_cause` varchar(128) DEFAULT NULL COMMENT '可能原因',
  `specific_problem` varchar(256) DEFAULT NULL COMMENT '具体问题',
  `additional_text` varchar(256) DEFAULT NULL COMMENT '附加文本',
  `additional_information` text COMMENT '附加信息',
  `managed_object_instance` varchar(256) DEFAULT NULL COMMENT '管理对象实例',
  `event_time` datetime(3) DEFAULT NULL COMMENT '事件时间',
  `start_time` datetime(3) DEFAULT NULL COMMENT '告警开始时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '告警结束时间',
  `last_changed` datetime(3) DEFAULT NULL COMMENT '最后变更时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_alarm_ident` (`alarm_identifier`),
  KEY `idx_tr069_alarms_deleted_at` (`deleted_at`),
  KEY `idx_device_id` (`device_id`),
  KEY `idx_serial_number` (`serial_number`),
  KEY `idx_source` (`source`),
  KEY `idx_status` (`status`),
  KEY `idx_perceived_severity` (`perceived_severity`),
  KEY `idx_event_time` (`event_time`),
  KEY `idx_start_time` (`start_time`),
  KEY `idx_end_time` (`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 七、扩展字段说明

以下字段为系统内部使用，不在协议字段中：

| 字段 | 说明 |
|------|------|
| `DeviceID` | 关联设备表ID |
| `Source` | 告警来源类型（从路径解析） |
| `Status` | 告警状态（Active/Cleared） |
| `StartTime` | 告警开始时间（等于EventTime） |
| `EndTime` | 告警结束时间（清除时设置） |
| `LastChanged` | 最后变更时间 |

---

## 八、开发注意事项

1. **告警去重**: 使用 `alarm_identifier` 作为唯一键，相同ID的告警会更新而非插入

2. **批量写入**: 使用 GORM 的 `CreateInBatches(alarms, 100)`，每批100条

3. **QueuedEvent 忽略**: 队列事件数据为空，直接跳过

4. **时间字段**: 设备可能使用 `EventTime` 或 `AlarmRaisedTime`，解析时需兼容

5. **OUI/SerialNumber**: 优先使用告警数据中的值，为空时使用设备上下文的值

---

## 九、文件清单

| 文件 | 说明 |
|------|------|
| `model/tr069_alarm.go` | 告警模型定义 |
| `model/support_tr069_alarm.go` | 支持的告警类型模型 |
| `adapter/gorm_repo.go` | 数据库适配器（SyncAlarms） |
| `api/alarm.go` | API 接口 |
| `router/alarm.go` | 路由注册 |
| `initialize/gorm.go` | 数据库迁移 |

---

## 十、后续优化方向

1. **Redis 缓冲队列**: 高并发场景下使用 Redis Stream 缓冲告警数据
2. **告警聚合**: 相同类型告警聚合显示
3. **告警通知**: 集成邮件/短信/钉钉通知
4. **告警确认**: 支持人工确认告警
5. **告警抑制**: 支持告警抑制规则
