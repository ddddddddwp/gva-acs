# TR069-Management 插件设计文档

## 1. 插件概述

TR069-Management是一个专门集成到GVA框架中的插件，负责提供TR069设备的数据查询、管理和监控功能。它通过读取TR069-Adapter服务存储的数据库信息，为用户提供完整的CPE设备管理界面。

## 2. 插件结构

```
server/plugin/tr069-management/
├── plugin.go                    # 插件入口文件
├── api/                         # API控制器层
│   ├── enter.go                 # API组入口
│   ├── device_api.go            # 设备管理API
│   ├── parameter_api.go         # 参数管理API
│   ├── session_api.go           # 会话管理API
│   ├── log_api.go               # 日志查询API
│   └── statistics_api.go        # 统计分析API
├── service/                     # 服务层
│   ├── enter.go                 # 服务组入口
│   ├── device_service.go        # 设备管理服务
│   ├── parameter_service.go     # 参数管理服务
│   ├── session_service.go       # 会话管理服务
│   ├── log_service.go           # 日志查询服务
│   └── statistics_service.go    # 统计分析服务
├── router/                      # 路由层
│   ├── enter.go                 # 路由组入口
│   ├── device_router.go         # 设备路由
│   ├── parameter_router.go      # 参数路由
│   ├── session_router.go        # 会话路由
│   ├── log_router.go            # 日志路由
│   └── statistics_router.go     # 统计路由
├── model/                       # 数据模型层
│   ├── device.go                # 设备模型
│   ├── parameter.go             # 参数模型
│   ├── session.go               # 会话模型
│   ├── log.go                   # 日志模型
│   └── request/                 # 请求模型
│       ├── device_request.go    # 设备请求模型
│       ├── parameter_request.go # 参数请求模型
│       ├── session_request.go   # 会话请求模型
│       └── log_request.go       # 日志请求模型
├── initialize/                  # 初始化模块
│   ├── gorm.go                  # 数据库初始化
│   ├── router.go                # 路由初始化
│   ├── menu.go                  # 菜单初始化
│   └── api.go                   # API权限初始化
└── config/                      # 配置文件
    └── config.go                # 插件配置
```

## 3. 核心功能模块

### 3.1 设备管理模块

#### 3.1.1 设备API (api/device_api.go)

```go
package api

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/service"
)

type DeviceApi struct{}

var deviceService = service.ServiceGroupApp.DeviceService

// GetDeviceList 获取设备列表
// @Tags     TR069设备管理
// @Summary  获取CPE设备列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.DeviceSearchRequest true "设备查询参数"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router   /tr069/device/getDeviceList [post]
func (d *DeviceApi) GetDeviceList(c *gin.Context) {
    var req request.DeviceSearchRequest
    err := c.ShouldBindJSON(&req)
    if err != nil {
        response.FailWithMessage(err.Error(), c)
        return
    }
    
    list, total, err := deviceService.GetDeviceList(req)
    if err != nil {
        global.GVA_LOG.Error("获取设备列表失败!", zap.Error(err))
        response.FailWithMessage("获取设备列表失败", c)
        return
    }
    
    response.OkWithDetailed(response.PageResult{
        List:     list,
        Total:    total,
        Page:     req.Page,
        PageSize: req.PageSize,
    }, "获取成功", c)
}

// GetDeviceDetail 获取设备详情
// @Tags     TR069设备管理
// @Summary  获取CPE设备详细信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    deviceId path string true "设备ID"
// @Success  200  {object} response.Response{data=model.Device,msg=string} "获取成功"
// @Router   /tr069/device/getDeviceDetail/{deviceId} [get]
func (d *DeviceApi) GetDeviceDetail(c *gin.Context) {
    deviceId := c.Param("deviceId")
    if deviceId == "" {
        response.FailWithMessage("设备ID不能为空", c)
        return
    }
    
    device, err := deviceService.GetDeviceDetail(deviceId)
    if err != nil {
        global.GVA_LOG.Error("获取设备详情失败!", zap.Error(err))
        response.FailWithMessage("获取设备详情失败", c)
        return
    }
    
    response.OkWithDetailed(device, "获取成功", c)
}

// UpdateDeviceConfig 更新设备配置
// @Tags     TR069设备管理
// @Summary  更新CPE设备配置信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.DeviceConfigRequest true "设备配置参数"
// @Success  200  {object} response.Response{msg=string} "更新成功"
// @Router   /tr069/device/updateConfig [put]
func (d *DeviceApi) UpdateDeviceConfig(c *gin.Context) {
    var req request.DeviceConfigRequest
    err := c.ShouldBindJSON(&req)
    if err != nil {
        response.FailWithMessage(err.Error(), c)
        return
    }
    
    err = deviceService.UpdateDeviceConfig(req)
    if err != nil {
        global.GVA_LOG.Error("更新设备配置失败!", zap.Error(err))
        response.FailWithMessage("更新设备配置失败", c)
        return
    }
    
    response.OkWithMessage("更新成功", c)
}

// GetDeviceStatistics 获取设备统计信息
// @Tags     TR069设备管理
// @Summary  获取设备统计信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Success  200  {object} response.Response{data=model.DeviceStatistics,msg=string} "获取成功"
// @Router   /tr069/device/getStatistics [get]
func (d *DeviceApi) GetDeviceStatistics(c *gin.Context) {
    statistics, err := deviceService.GetDeviceStatistics()
    if err != nil {
        global.GVA_LOG.Error("获取设备统计信息失败!", zap.Error(err))
        response.FailWithMessage("获取设备统计信息失败", c)
        return
    }
    
    response.OkWithDetailed(statistics, "获取成功", c)
}
```

#### 3.1.2 设备服务 (service/device_service.go)

```go
package service

import (
    "errors"
    "time"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
    "gorm.io/gorm"
)

type DeviceService struct{}

// GetDeviceList 获取设备列表
func (d *DeviceService) GetDeviceList(req request.DeviceSearchRequest) (list []model.Device, total int64, err error) {
    limit := req.PageSize
    offset := req.PageSize * (req.Page - 1)
    
    db := global.GVA_DB.Model(&model.Device{})
    
    // 构建查询条件
    if req.DeviceID != "" {
        db = db.Where("device_id LIKE ?", "%"+req.DeviceID+"%")
    }
    if req.Manufacturer != "" {
        db = db.Where("manufacturer LIKE ?", "%"+req.Manufacturer+"%")
    }
    if req.ProductClass != "" {
        db = db.Where("product_class LIKE ?", "%"+req.ProductClass+"%")
    }
    if req.OnlineStatus != nil {
        db = db.Where("online_status = ?", *req.OnlineStatus)
    }
    if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
        db = db.Where("last_inform_time BETWEEN ? AND ?", req.StartTime, req.EndTime)
    }
    
    // 获取总数
    err = db.Count(&total).Error
    if err != nil {
        return
    }
    
    // 获取列表数据
    err = db.Limit(limit).Offset(offset).Order("last_inform_time DESC").Find(&list).Error
    return
}

// GetDeviceDetail 获取设备详情
func (d *DeviceService) GetDeviceDetail(deviceId string) (device model.Device, err error) {
    err = global.GVA_DB.Where("device_id = ?", deviceId).First(&device).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return device, errors.New("设备不存在")
        }
        return device, err
    }
    return
}

// UpdateDeviceConfig 更新设备配置
func (d *DeviceService) UpdateDeviceConfig(req request.DeviceConfigRequest) error {
    var device model.Device
    err := global.GVA_DB.Where("device_id = ?", req.DeviceID).First(&device).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errors.New("设备不存在")
        }
        return err
    }
    
    // 更新设备配置信息
    updates := map[string]interface{}{
        "connection_request_url": req.ConnectionRequestURL,
        "updated_at":            time.Now(),
    }
    
    return global.GVA_DB.Model(&device).Updates(updates).Error
}

// GetDeviceStatistics 获取设备统计信息
func (d *DeviceService) GetDeviceStatistics() (statistics model.DeviceStatistics, err error) {
    // 总设备数
    err = global.GVA_DB.Model(&model.Device{}).Count(&statistics.TotalDevices).Error
    if err != nil {
        return
    }
    
    // 在线设备数
    err = global.GVA_DB.Model(&model.Device{}).Where("online_status = ?", 1).Count(&statistics.OnlineDevices).Error
    if err != nil {
        return
    }
    
    // 离线设备数
    statistics.OfflineDevices = statistics.TotalDevices - statistics.OnlineDevices
    
    // 按厂商统计
    var manufacturerStats []model.ManufacturerStat
    err = global.GVA_DB.Model(&model.Device{}).
        Select("manufacturer, COUNT(*) as count").
        Group("manufacturer").
        Find(&manufacturerStats).Error
    if err != nil {
        return
    }
    statistics.ManufacturerStats = manufacturerStats
    
    // 按产品类别统计
    var productStats []model.ProductStat
    err = global.GVA_DB.Model(&model.Device{}).
        Select("product_class, COUNT(*) as count").
        Group("product_class").
        Find(&productStats).Error
    if err != nil {
        return
    }
    statistics.ProductStats = productStats
    
    return
}

// GetDeviceOnlineHistory 获取设备在线历史
func (d *DeviceService) GetDeviceOnlineHistory(deviceId string, days int) (history []model.OnlineHistory, err error) {
    startTime := time.Now().AddDate(0, 0, -days)
    
    err = global.GVA_DB.Raw(`
        SELECT 
            DATE(created_at) as date,
            COUNT(CASE WHEN operation_type = 'online' THEN 1 END) as online_count,
            COUNT(CASE WHEN operation_type = 'offline' THEN 1 END) as offline_count
        FROM cpe_operation_logs 
        WHERE device_id = ? AND created_at >= ? 
        GROUP BY DATE(created_at)
        ORDER BY date
    `, deviceId, startTime).Scan(&history).Error
    
    return
}
```

### 3.2 参数管理模块

#### 3.2.1 参数API (api/parameter_api.go)

```go
package api

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/service"
)

type ParameterApi struct{}

var parameterService = service.ServiceGroupApp.ParameterService

// GetParameterTree 获取设备参数树
// @Tags     TR069参数管理
// @Summary  获取CPE设备参数树结构
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    deviceId path string true "设备ID"
// @Success  200  {object} response.Response{data=[]model.ParameterNode,msg=string} "获取成功"
// @Router   /tr069/parameter/getParameterTree/{deviceId} [get]
func (p *ParameterApi) GetParameterTree(c *gin.Context) {
    deviceId := c.Param("deviceId")
    if deviceId == "" {
        response.FailWithMessage("设备ID不能为空", c)
        return
    }
    
    tree, err := parameterService.GetParameterTree(deviceId)
    if err != nil {
        global.GVA_LOG.Error("获取参数树失败!", zap.Error(err))
        response.FailWithMessage("获取参数树失败", c)
        return
    }
    
    response.OkWithDetailed(tree, "获取成功", c)
}

// GetParameterList 获取参数列表
// @Tags     TR069参数管理
// @Summary  获取CPE设备参数列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.ParameterSearchRequest true "参数查询参数"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router   /tr069/parameter/getParameterList [post]
func (p *ParameterApi) GetParameterList(c *gin.Context) {
    var req request.ParameterSearchRequest
    err := c.ShouldBindJSON(&req)
    if err != nil {
        response.FailWithMessage(err.Error(), c)
        return
    }
    
    list, total, err := parameterService.GetParameterList(req)
    if err != nil {
        global.GVA_LOG.Error("获取参数列表失败!", zap.Error(err))
        response.FailWithMessage("获取参数列表失败", c)
        return
    }
    
    response.OkWithDetailed(response.PageResult{
        List:     list,
        Total:    total,
        Page:     req.Page,
        PageSize: req.PageSize,
    }, "获取成功", c)
}

// BatchSetParameters 批量设置参数
// @Tags     TR069参数管理
// @Summary  批量设置CPE设备参数
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.BatchParameterRequest true "批量参数设置"
// @Success  200  {object} response.Response{msg=string} "设置成功"
// @Router   /tr069/parameter/batchSet [post]
func (p *ParameterApi) BatchSetParameters(c *gin.Context) {
    var req request.BatchParameterRequest
    err := c.ShouldBindJSON(&req)
    if err != nil {
        response.FailWithMessage(err.Error(), c)
        return
    }
    
    err = parameterService.BatchSetParameters(req)
    if err != nil {
        global.GVA_LOG.Error("批量设置参数失败!", zap.Error(err))
        response.FailWithMessage("批量设置参数失败", c)
        return
    }
    
    response.OkWithMessage("设置成功", c)
}

// GetParameterHistory 获取参数历史
// @Tags     TR069参数管理
// @Summary  获取参数变化历史
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.ParameterHistoryRequest true "参数历史查询"
// @Success  200  {object} response.Response{data=[]model.ParameterHistory,msg=string} "获取成功"
// @Router   /tr069/parameter/getHistory [post]
func (p *ParameterApi) GetParameterHistory(c *gin.Context) {
    var req request.ParameterHistoryRequest
    err := c.ShouldBindJSON(&req)
    if err != nil {
        response.FailWithMessage(err.Error(), c)
        return
    }
    
    history, err := parameterService.GetParameterHistory(req)
    if err != nil {
        global.GVA_LOG.Error("获取参数历史失败!", zap.Error(err))
        response.FailWithMessage("获取参数历史失败", c)
        return
    }
    
    response.OkWithDetailed(history, "获取成功", c)
}
```

### 3.3 数据模型定义

#### 3.3.1 设备模型 (model/device.go)

```go
package model

import (
    "time"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
)

// Device CPE设备模型
type Device struct {
    global.GVA_MODEL
    DeviceID             string    `json:"device_id" gorm:"uniqueIndex;size:255;not null;comment:设备ID"`
    Manufacturer         string    `json:"manufacturer" gorm:"size:100;comment:制造商"`
    OUI                  string    `json:"oui" gorm:"size:6;comment:组织唯一标识符"`
    ProductClass         string    `json:"product_class" gorm:"size:100;comment:产品类别"`
    SerialNumber         string    `json:"serial_number" gorm:"size:100;comment:序列号"`
    HardwareVersion      string    `json:"hardware_version" gorm:"size:50;comment:硬件版本"`
    SoftwareVersion      string    `json:"software_version" gorm:"size:50;comment:软件版本"`
    ConnectionRequestURL string    `json:"connection_request_url" gorm:"size:500;comment:连接请求URL"`
    LastInformTime       time.Time `json:"last_inform_time" gorm:"comment:最后通信时间"`
    OnlineStatus         int       `json:"online_status" gorm:"default:0;comment:在线状态 0:离线 1:在线"`
}

func (Device) TableName() string {
    return "cpe_devices"
}

// DeviceStatistics 设备统计信息
type DeviceStatistics struct {
    TotalDevices      int64              `json:"total_devices"`      // 总设备数
    OnlineDevices     int64              `json:"online_devices"`     // 在线设备数
    OfflineDevices    int64              `json:"offline_devices"`    // 离线设备数
    ManufacturerStats []ManufacturerStat `json:"manufacturer_stats"` // 厂商统计
    ProductStats      []ProductStat      `json:"product_stats"`      // 产品统计
}

// ManufacturerStat 厂商统计
type ManufacturerStat struct {
    Manufacturer string `json:"manufacturer"`
    Count        int64  `json:"count"`
}

// ProductStat 产品统计
type ProductStat struct {
    ProductClass string `json:"product_class"`
    Count        int64  `json:"count"`
}

// OnlineHistory 在线历史
type OnlineHistory struct {
    Date         string `json:"date"`
    OnlineCount  int    `json:"online_count"`
    OfflineCount int    `json:"offline_count"`
}
```

#### 3.3.2 参数模型 (model/parameter.go)

```go
package model

import (
    "time"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
)

// Parameter 参数模型
type Parameter struct {
    global.GVA_MODEL
    DeviceID       string    `json:"device_id" gorm:"index;size:255;not null;comment:设备ID"`
    ParameterName  string    `json:"parameter_name" gorm:"size:500;not null;comment:参数名称"`
    ParameterValue string    `json:"parameter_value" gorm:"type:text;comment:参数值"`
    ParameterType  string    `json:"parameter_type" gorm:"size:50;comment:参数类型"`
    Writable       bool      `json:"writable" gorm:"default:false;comment:是否可写"`
    LastUpdated    time.Time `json:"last_updated" gorm:"comment:最后更新时间"`
}

func (Parameter) TableName() string {
    return "cpe_parameters"
}

// ParameterNode 参数树节点
type ParameterNode struct {
    Name     string           `json:"name"`
    FullPath string           `json:"full_path"`
    Value    string           `json:"value,omitempty"`
    Type     string           `json:"type,omitempty"`
    Writable bool             `json:"writable"`
    Children []*ParameterNode `json:"children,omitempty"`
}

// ParameterHistory 参数历史
type ParameterHistory struct {
    ID            uint      `json:"id"`
    DeviceID      string    `json:"device_id"`
    ParameterName string    `json:"parameter_name"`
    OldValue      string    `json:"old_value"`
    NewValue      string    `json:"new_value"`
    ChangeTime    time.Time `json:"change_time"`
    ChangeReason  string    `json:"change_reason"`
}
```

### 3.4 前端页面设计

#### 3.4.1 设备列表页面结构

```
web/src/view/tr069-management/
├── device/
│   ├── index.vue                # 设备列表主页面
│   ├── detail.vue               # 设备详情页面
│   ├── config.vue               # 设备配置页面
│   └── components/
│       ├── DeviceTable.vue      # 设备表格组件
│       ├── DeviceFilter.vue     # 设备筛选组件
│       ├── DeviceChart.vue      # 设备统计图表
│       └── DeviceStatus.vue     # 设备状态组件
├── parameter/
│   ├── index.vue                # 参数管理主页面
│   ├── tree.vue                 # 参数树页面
│   ├── batch.vue                # 批量设置页面
│   └── components/
│       ├── ParameterTree.vue    # 参数树组件
│       ├── ParameterTable.vue   # 参数表格组件
│       ├── ParameterEditor.vue  # 参数编辑器
│       └── ParameterHistory.vue # 参数历史组件
├── session/
│   ├── index.vue                # 会话管理页面
│   └── components/
│       ├── SessionTable.vue     # 会话表格组件
│       └── SessionDetail.vue    # 会话详情组件
├── log/
│   ├── index.vue                # 日志查询页面
│   └── components/
│       ├── LogTable.vue         # 日志表格组件
│       └── LogFilter.vue        # 日志筛选组件
└── dashboard/
    ├── index.vue                # 监控仪表盘
    └── components/
        ├── DeviceOverview.vue   # 设备概览组件
        ├── OnlineChart.vue      # 在线趋势图
        ├── AlarmList.vue        # 告警列表
        └── StatisticsCard.vue   # 统计卡片
```

#### 3.4.2 设备列表页面示例 (web/src/view/tr069-management/device/index.vue)

```vue
<template>
  <div class="device-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-title">
        <h2>CPE设备管理</h2>
        <p>管理和监控所有TR069 CPE设备</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="success" @click="exportData">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="statistics-cards">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon total">
                <el-icon><Monitor /></el-icon>
              </div>
              <div class="stat-info">
                <h3>{{ statistics.totalDevices }}</h3>
                <p>总设备数</p>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon online">
                <el-icon><CircleCheck /></el-icon>
              </div>
              <div class="stat-info">
                <h3>{{ statistics.onlineDevices }}</h3>
                <p>在线设备</p>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon offline">
                <el-icon><CircleClose /></el-icon>
              </div>
              <div class="stat-info">
                <h3>{{ statistics.offlineDevices }}</h3>
                <p>离线设备</p>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stat-card">
            <div class="stat-content">
              <div class="stat-icon rate">
                <el-icon><TrendCharts /></el-icon>
              </div>
              <div class="stat-info">
                <h3>{{ onlineRate }}%</h3>
                <p>在线率</p>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 搜索筛选 -->
    <el-card class="search-card">
      <el-form :model="searchForm" :inline="true" label-width="80px">
        <el-form-item label="设备ID">
          <el-input 
            v-model="searchForm.deviceId" 
            placeholder="请输入设备ID"
            clearable
          />
        </el-form-item>
        <el-form-item label="制造商">
          <el-select 
            v-model="searchForm.manufacturer" 
            placeholder="请选择制造商"
            clearable
          >
            <el-option 
              v-for="item in manufacturerOptions" 
              :key="item.value" 
              :label="item.label" 
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="在线状态">
          <el-select 
            v-model="searchForm.onlineStatus" 
            placeholder="请选择状态"
            clearable
          >
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="searchForm.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="resetSearch">
            <el-icon><RefreshRight /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 设备表格 -->
    <el-card class="table-card">
      <el-table 
        :data="deviceList" 
        v-loading="loading"
        stripe
        border
        style="width: 100%"
      >
        <el-table-column prop="deviceId" label="设备ID" width="200" />
        <el-table-column prop="manufacturer" label="制造商" width="120" />
        <el-table-column prop="productClass" label="产品类别" width="150" />
        <el-table-column prop="serialNumber" label="序列号" width="150" />
        <el-table-column prop="softwareVersion" label="软件版本" width="120" />
        <el-table-column label="在线状态" width="100">
          <template #default="{ row }">
            <el-tag 
              :type="row.onlineStatus === 1 ? 'success' : 'danger'"
              size="small"
            >
              {{ row.onlineStatus === 1 ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastInformTime" label="最后通信时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.lastInformTime) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="primary" 
              size="small" 
              @click="viewDetail(row)"
            >
              详情
            </el-button>
            <el-button 
              type="success" 
              size="small" 
              @click="manageParameters(row)"
            >
              参数
            </el-button>
            <el-button 
              type="warning" 
              size="small" 
              @click="viewLogs(row)"
            >
              日志
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  getDeviceList, 
  getDeviceStatistics 
} from '@/api/tr069-management/device'

// 路由
const router = useRouter()

// 响应式数据
const loading = ref(false)
const deviceList = ref([])
const statistics = ref({
  totalDevices: 0,
  onlineDevices: 0,
  offlineDevices: 0
})

// 搜索表单
const searchForm = reactive({
  deviceId: '',
  manufacturer: '',
  onlineStatus: null,
  timeRange: []
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 制造商选项
const manufacturerOptions = ref([])

// 计算属性
const onlineRate = computed(() => {
  if (statistics.value.totalDevices === 0) return 0
  return Math.round((statistics.value.onlineDevices / statistics.value.totalDevices) * 100)
})

// 方法
const fetchDeviceList = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      deviceId: searchForm.deviceId,
      manufacturer: searchForm.manufacturer,
      onlineStatus: searchForm.onlineStatus,
      startTime: searchForm.timeRange?.[0],
      endTime: searchForm.timeRange?.[1]
    }
    
    const res = await getDeviceList(params)
    if (res.code === 0) {
      deviceList.value = res.data.list
      pagination.total = res.data.total
    }
  } catch (error) {
    ElMessage.error('获取设备列表失败')
  } finally {
    loading.value = false
  }
}

const fetchStatistics = async () => {
  try {
    const res = await getDeviceStatistics()
    if (res.code === 0) {
      statistics.value = res.data
      
      // 提取制造商选项
      manufacturerOptions.value = res.data.manufacturerStats.map(item => ({
        label: item.manufacturer,
        value: item.manufacturer
      }))
    }
  } catch (error) {
    ElMessage.error('获取统计信息失败')
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchDeviceList()
}

const resetSearch = () => {
  Object.assign(searchForm, {
    deviceId: '',
    manufacturer: '',
    onlineStatus: null,
    timeRange: []
  })
  handleSearch()
}

const handleSizeChange = (size) => {
  pagination.pageSize = size
  fetchDeviceList()
}

const handleCurrentChange = (page) => {
  pagination.page = page
  fetchDeviceList()
}

const refreshData = () => {
  fetchDeviceList()
  fetchStatistics()
}

const viewDetail = (row) => {
  router.push(`/tr069-management/device/detail/${row.deviceId}`)
}

const manageParameters = (row) => {
  router.push(`/tr069-management/parameter/tree/${row.deviceId}`)
}

const viewLogs = (row) => {
  router.push(`/tr069-management/log?deviceId=${row.deviceId}`)
}

const exportData = () => {
  ElMessage.info('导出功能开发中...')
}

const formatTime = (time) => {
  return new Date(time).toLocaleString()
}

// 生命周期
onMounted(() => {
  fetchDeviceList()
  fetchStatistics()
})
</script>

<style scoped>
.device-management {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-title h2 {
  margin: 0;
  color: #303133;
}

.header-title p {
  margin: 5px 0 0 0;
  color: #909399;
  font-size: 14px;
}

.statistics-cards {
  margin-bottom: 20px;
}

.stat-card {
  border: none;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 15px;
  font-size: 24px;
  color: white;
}

.stat-icon.total { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.stat-icon.online { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.stat-icon.offline { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.stat-icon.rate { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }

.stat-info h3 {
  margin: 0;
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-info p {
  margin: 5px 0 0 0;
  color: #909399;
  font-size: 14px;
}

.search-card, .table-card {
  margin-bottom: 20px;
  border: none;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
```

## 4. 插件初始化

### 4.1 路由初始化 (initialize/router.go)

```go
package initialize

import (
    "github.com/gin-gonic/gin"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/router"
)

func InitializeRouter(Router *gin.RouterGroup) {
    tr069Router := Router.Group("tr069")
    {
        router.RouterGroupApp.DeviceRouter.InitDeviceRouter(tr069Router)
        router.RouterGroupApp.ParameterRouter.InitParameterRouter(tr069Router)
        router.RouterGroupApp.SessionRouter.InitSessionRouter(tr069Router)
        router.RouterGroupApp.LogRouter.InitLogRouter(tr069Router)
        router.RouterGroupApp.StatisticsRouter.InitStatisticsRouter(tr069Router)
    }
}
```

### 4.2 菜单初始化 (initialize/menu.go)

```go
package initialize

import (
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

func InitializeMenu() error {
    // TR069管理主菜单
    tr069Menu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "0",
        Path:      "tr069-management",
        Name:      "tr069Management",
        Hidden:    false,
        Component: "view/tr069-management/index.vue",
        Sort:      4,
        Meta: system.Meta{
            Title:       "TR069管理",
            Icon:        "monitor",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 设备管理菜单
    deviceMenu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "tr069-management",
        Path:      "device",
        Name:      "deviceManagement",
        Hidden:    false,
        Component: "view/tr069-management/device/index.vue",
        Sort:      1,
        Meta: system.Meta{
            Title:       "设备管理",
            Icon:        "cpu",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 参数管理菜单
    parameterMenu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "tr069-management",
        Path:      "parameter",
        Name:      "parameterManagement",
        Hidden:    false,
        Component: "view/tr069-management/parameter/index.vue",
        Sort:      2,
        Meta: system.Meta{
            Title:       "参数管理",
            Icon:        "setting",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 会话管理菜单
    sessionMenu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "tr069-management",
        Path:      "session",
        Name:      "sessionManagement",
        Hidden:    false,
        Component: "view/tr069-management/session/index.vue",
        Sort:      3,
        Meta: system.Meta{
            Title:       "会话管理",
            Icon:        "connection",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 日志查询菜单
    logMenu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "tr069-management",
        Path:      "log",
        Name:      "logQuery",
        Hidden:    false,
        Component: "view/tr069-management/log/index.vue",
        Sort:      4,
        Meta: system.Meta{
            Title:       "日志查询",
            Icon:        "document",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 监控仪表盘菜单
    dashboardMenu := system.SysBaseMenu{
        MenuLevel: 0,
        ParentId:  "tr069-management",
        Path:      "dashboard",
        Name:      "tr069Dashboard",
        Hidden:    false,
        Component: "view/tr069-management/dashboard/index.vue",
        Sort:      5,
        Meta: system.Meta{
            Title:       "监控仪表盘",
            Icon:        "data-analysis",
            KeepAlive:   true,
            DefaultMenu: false,
        },
    }
    
    // 创建菜单
    menus := []system.SysBaseMenu{
        tr069Menu,
        deviceMenu,
        parameterMenu,
        sessionMenu,
        logMenu,
        dashboardMenu,
    }
    
    for _, menu := range menus {
        if err := global.GVA_DB.Create(&menu).Error; err != nil {
            return err
        }
    }
    
    return nil
}
```

## 5. 部署和集成

### 5.1 插件入口 (plugin.go)

```go
package tr069_management

import (
    "github.com/gin-gonic/gin"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/initialize"
)

type TR069ManagementPlugin struct{}

func (p *TR069ManagementPlugin) Register(group *gin.RouterGroup) {
    initialize.InitializeRouter(group)
}

func (p *TR069ManagementPlugin) RouterPath() string {
    return "tr069-management"
}

func CreateTR069ManagementPlug() *TR069ManagementPlugin {
    return &TR069ManagementPlugin{}
}
```

### 5.2 前端API接口 (web/src/api/tr069-management/device.js)

```javascript
import service from '@/utils/request'

// 获取设备列表
export const getDeviceList = (data) => {
  return service({
    url: '/tr069/device/getDeviceList',
    method: 'post',
    data: data
  })
}

// 获取设备详情
export const getDeviceDetail = (deviceId) => {
  return service({
    url: `/tr069/device/getDeviceDetail/${deviceId}`,
    method: 'get'
  })
}

// 更新设备配置
export const updateDeviceConfig = (data) => {
  return service({
    url: '/tr069/device/updateConfig',
    method: 'put',
    data: data
  })
}

// 获取设备统计信息
export const getDeviceStatistics = () => {
  return service({
    url: '/tr069/device/getStatistics',
    method: 'get'
  })
}

// 获取设备在线历史
export const getDeviceOnlineHistory = (deviceId, days) => {
  return service({
    url: `/tr069/device/getOnlineHistory/${deviceId}/${days}`,
    method: 'get'
  })
}
```

---

这个TR069-Management插件设计文档详细描述了如何在GVA框架中创建一个完整的TR069设备管理插件，包括后端API、前端界面、数据模型和部署配置。插件专注于数据查询和用户交互，与TR069-Adapter服务形成完美的职责分离。