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
// @Summary 分页获取告警列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param serialNumber query string false "序列号"
// @Param status query string false "状态"
// @Param severity query string false "级别"
// @Router /tr069/alarm/list [get]
func (a *AlarmApi) GetAlarmList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)

	serialNumber := c.Query("serialNumber")
	status := c.Query("status")
	severity := c.Query("severity")

	db := global.GVA_DB.Model(&model.Tr069Alarm{})

	if serialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+serialNumber+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if severity != "" {
		db = db.Where("severity = ?", severity)
	}

	var list []model.Tr069Alarm
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		response.FailWithMessage("查询失败", c)
		return
	}

	// Paginate using the helper or manual
	// Manual is safer if we want to ensure ordering
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
