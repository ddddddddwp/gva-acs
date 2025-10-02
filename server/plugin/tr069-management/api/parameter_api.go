package api

import (
	"fmt"
	
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ParameterApi 参数API
type ParameterApi struct{}

// GetParametersByDevice 获取设备参数列表
// @Tags     TR069Management
// @Summary  获取设备参数列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data query request.ParameterSearch true "分页、搜索条件"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "参数列表,返回包括列表,总数,页码,每页数量"
// @Router   /tr069-management/parameter/list [get]
func (a *ParameterApi) GetParametersByDevice(c *gin.Context) {
	var req request.ParameterSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	parameters, total, err := parameterService.GetParametersByDevice(req)
	if err != nil {
		global.GVA_LOG.Error("获取参数列表失败", zap.Error(err))
		response.FailWithMessage("获取参数列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     parameters,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// GetParameterByID 根据ID获取参数详情
// @Tags     TR069Management
// @Summary  根据ID获取参数详情
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path uint true "参数ID"
// @Success  200  {object} response.Response{data=model.Parameter,msg=string} "参数详情"
// @Router   /tr069-management/parameter/{id} [get]
func (a *ParameterApi) GetParameterByID(c *gin.Context) {
	var id uint
	if err := c.ShouldBindUri(struct{ ID uint `uri:"id"` }{ID: id}); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	parameter, err := parameterService.GetParameterByID(id)
	if err != nil {
		global.GVA_LOG.Error("获取参数详情失败", zap.Error(err))
		response.FailWithMessage("获取参数详情失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(parameter, "获取成功", c)
}

// SetDeviceParameters 设置设备参数
// @Tags     TR069Management
// @Summary  设置设备参数
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.SetParametersRequest true "参数信息"
// @Success  200  {object} response.Response{msg=string} "设置成功"
// @Router   /tr069-management/parameter/set [post]
func (a *ParameterApi) SetDeviceParameters(c *gin.Context) {
	var req request.SetParametersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	err := parameterService.SetDeviceParameters(req)
	if err != nil {
		global.GVA_LOG.Error("设置参数失败", zap.Error(err))
		response.FailWithMessage("设置参数失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("设置成功", c)
}

// GetParameterCategories 获取参数分类列表
// @Tags     TR069Management
// @Summary  获取参数分类列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    deviceId query uint true "设备ID"
// @Success  200  {object} response.Response{data=[]string,msg=string} "分类列表"
// @Router   /tr069-management/parameter/categories [get]
func (a *ParameterApi) GetParameterCategories(c *gin.Context) {
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		response.FailWithMessage("设备ID不能为空", c)
		return
	}

	var deviceIDUint uint
	if _, err := fmt.Sscanf(deviceID, "%d", &deviceIDUint); err != nil {
		response.FailWithMessage("设备ID格式错误", c)
		return
	}

	categories, err := parameterService.GetParameterCategories(deviceIDUint)
	if err != nil {
		global.GVA_LOG.Error("获取参数分类列表失败", zap.Error(err))
		response.FailWithMessage("获取参数分类列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(categories, "获取成功", c)
}