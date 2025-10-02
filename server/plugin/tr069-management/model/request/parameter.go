package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// ParameterSearch 参数搜索请求
type ParameterSearch struct {
	request.PageInfo
	DeviceID uint   `json:"deviceId" form:"deviceId"`
	Name     string `json:"name" form:"name"`
	Category string `json:"category" form:"category"`
	Tags     string `json:"tags" form:"tags"`
}

// SetParametersRequest 设置参数请求
type SetParametersRequest struct {
	DeviceID   uint                   `json:"deviceId" binding:"required"`
	Parameters []ParameterValueUpdate `json:"parameters" binding:"required"`
}

// ParameterValueUpdate 参数值更新
type ParameterValueUpdate struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
	Type  string `json:"type"`
}