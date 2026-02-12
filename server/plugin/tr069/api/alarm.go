package api

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/gin-gonic/gin"
)

type AlarmApi struct{}

// GetAlarmList
// @Tags TR069
// @Summary Get Active Alarm List (Default)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "Page Number"
// @Param pageSize query int true "Page Size"
// @Param serialNumber query string false "Serial Number"
// @Param source query string false "Source (CurrentAlarm/ExpeditedEvent/HistoryEvent)"
// @Param severity query string false "Severity"
// @Router /tr069/alarm/list [get]
func (a *AlarmApi) GetAlarmList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)

	serialNumber := c.Query("serialNumber")
	source := c.Query("source")
	severity := c.Query("severity")

	db := global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ?", "Active")

	if serialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+serialNumber+"%")
	}
	if source != "" {
		db = db.Where("source = ?", source)
	}
	if severity != "" {
		db = db.Where("perceived_severity = ?", severity)
	}

	var list []model.Tr069Alarm
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}

	limit := pageInfo.PageSize
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	page := pageInfo.Page
	if page <= 0 {
		page = 1
	}

	offset := limit * (page - 1)

	err = db.Limit(limit).Offset(offset).Order("start_time desc").Find(&list).Error
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: limit,
	}, "获取成功", c)
}

// GetHistoryAlarms
// @Tags TR069
// @Summary Get History Alarm List (Cleared)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "Page Number"
// @Param pageSize query int true "Page Size"
// @Param serialNumber query string false "Serial Number"
// @Param source query string false "Source (CurrentAlarm/ExpeditedEvent/HistoryEvent)"
// @Param severity query string false "Severity"
// @Router /tr069/alarm/history [get]
func (a *AlarmApi) GetHistoryAlarms(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)

	serialNumber := c.Query("serialNumber")
	source := c.Query("source")
	severity := c.Query("severity")

	db := global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ?", "Cleared")

	if serialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+serialNumber+"%")
	}
	if source != "" {
		db = db.Where("source = ?", source)
	}
	if severity != "" {
		db = db.Where("perceived_severity = ?", severity)
	}

	var list []model.Tr069Alarm
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}

	limit := pageInfo.PageSize
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	page := pageInfo.Page
	if page <= 0 {
		page = 1
	}

	offset := limit * (page - 1)

	err = db.Limit(limit).Offset(offset).Order("end_time desc").Find(&list).Error
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: limit,
	}, "获取成功", c)
}

// GetAlarmStats
// @Tags TR069
// @Summary Get Alarm Stats
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Router /tr069/alarm/stats [get]
func (a *AlarmApi) GetAlarmStats(c *gin.Context) {
	type Stats struct {
		Total    int64 `json:"total"`
		Active   int64 `json:"active"`
		Cleared  int64 `json:"cleared"`
		Critical int64 `json:"critical"`
		Major    int64 `json:"major"`
		Minor    int64 `json:"minor"`
		Warning  int64 `json:"warning"`
	}

	var stats Stats

	global.GVA_DB.Model(&model.Tr069Alarm{}).Count(&stats.Total)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ?", "Active").Count(&stats.Active)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ?", "Cleared").Count(&stats.Cleared)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ? AND perceived_severity = ?", "Active", "Critical").Count(&stats.Critical)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ? AND perceived_severity = ?", "Active", "Major").Count(&stats.Major)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ? AND perceived_severity = ?", "Active", "Minor").Count(&stats.Minor)
	global.GVA_DB.Model(&model.Tr069Alarm{}).Where("status = ? AND perceived_severity = ?", "Active", "Warning").Count(&stats.Warning)

	response.OkWithDetailed(stats, "获取成功", c)
}
