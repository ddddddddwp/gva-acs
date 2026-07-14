package api

import (
	"strconv"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type DataModelApi struct{}

var dmService = new(service.CommandService)

// FullSync
// @Tags TR069
// @Summary 同步设备参数(立即下发 Device. GetParameterValues)
// @Description 立即下发一次参数路径固定为 Device. 的 GetParameterValues 命令，并触发 Connection Request 主动连接设备
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/datamodel/{deviceId}/sync [post]
func (a *DataModelApi) FullSync(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	commandID, err := dmService.EnqueueDeviceParameterSync(uint(deviceId))
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(map[string]string{"commandId": commandID}, "任务已下发，等待设备响应", c)
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
