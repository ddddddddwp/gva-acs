package api

import (
	"strconv"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type DataModelApi struct{}

var dmService = new(service.CommandService)

type FullSyncRequest struct {
	MaxDepth int `json:"maxDepth" form:"maxDepth"`
}

// FullSync
// @Tags TR069
// @Summary 全量同步数据模型(递归 GetParameterNames + GetParameterValues)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body api.FullSyncRequest false "同步参数"
// @Success 200 {object} response.Response{msg=string} "下发成功"
// @Router /tr069/datamodel/{deviceId}/sync [post]
func (a *DataModelApi) FullSync(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	var in FullSyncRequest
	_ = c.ShouldBindJSON(&in)
	if err := dmService.EnqueueFullDataModelSync(uint(deviceId), in.MaxDepth); err != nil {
		response.FailWithMessage("任务下发失败", c)
		return
	}
	response.OkWithMessage("任务已下发，设备下次会话会开始上报", c)
}

// GetDataModelList
// @Tags TR069
// @Summary 查询设备已同步的数据模型列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param page query int false "页码"
// @Param pageSize query int false "每页大小"
// @Param prefix query string false "参数名前缀"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /tr069/datamodel/{deviceId}/list [get]
func (a *DataModelApi) GetDataModelList(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	prefix := c.Query("prefix")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	svc := new(service.DataModelService)
	list, total, err := svc.GetDataModelList(uint(deviceId), prefix, (page-1)*pageSize, pageSize)
	if err != nil {
		response.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

// GetDataModelStructure
// @Tags TR069
// @Summary 查询设备的数据模型结构（仅对象路径）
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{data=[]string,msg=string} "获取成功"
// @Router /tr069/datamodel/{deviceId}/structure [get]
func (a *DataModelApi) GetDataModelStructure(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	svc := new(service.DataModelService)
	list, err := svc.GetDataModelStructure(uint(deviceId))
	if err != nil {
		response.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "获取成功", c)
}
