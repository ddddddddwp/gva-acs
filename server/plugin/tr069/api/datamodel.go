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

