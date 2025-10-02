package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// ParameterSearch 参数搜索请求
type ParameterSearch struct {
	request.PageInfo
	DeviceID uint   `json:"deviceId" form:"deviceId"`
	Name     string `json:"name" form:"name"`
	Type     string `json:"type" form:"type"`
	Writable *bool  `json:"writable" form:"writable"`
}

// GetParameterRequest 获取参数请求
type GetParameterRequest struct {
	DeviceID        uint     `json:"deviceId" binding:"required"`
	ParameterNames  []string `json:"parameterNames" binding:"required,min=1"`
}

// SetParameterRequest 设置参数请求
type SetParameterRequest struct {
	DeviceID   uint                   `json:"deviceId" binding:"required"`
	Parameters []ParameterValuePair   `json:"parameters" binding:"required,min=1"`
}

// ParameterValuePair 参数名值对
type ParameterValuePair struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// AddObjectRequest 添加对象请求
type AddObjectRequest struct {
	DeviceID   uint   `json:"deviceId" binding:"required"`
	ObjectName string `json:"objectName" binding:"required"`
}

// DeleteObjectRequest 删除对象请求
type DeleteObjectRequest struct {
	DeviceID   uint   `json:"deviceId" binding:"required"`
	ObjectName string `json:"objectName" binding:"required"`
}

// GetParameterNamesRequest 获取参数名称请求
type GetParameterNamesRequest struct {
	DeviceID      uint   `json:"deviceId" binding:"required"`
	ParameterPath string `json:"parameterPath"`
	NextLevel     bool   `json:"nextLevel"`
}

// GetParameterAttributesRequest 获取参数属性请求
type GetParameterAttributesRequest struct {
	DeviceID   uint     `json:"deviceId" binding:"required"`
	Parameters []string `json:"parameters" binding:"required,min=1"`
}

// SetParameterAttributesRequest 设置参数属性请求
type SetParameterAttributesRequest struct {
	DeviceID   uint                      `json:"deviceId" binding:"required"`
	Parameters []ParameterAttributePair  `json:"parameters" binding:"required,min=1"`
}

// ParameterAttributePair 参数属性对
type ParameterAttributePair struct {
	Name         string `json:"name" binding:"required"`
	Notification int    `json:"notification"`
	AccessList   string `json:"accessList"`
}

// SetParametersRequest 批量设置参数请求
type SetParametersRequest struct {
	DeviceID   uint                   `json:"deviceId" binding:"required"`
	Parameters []ParameterValuePair   `json:"parameters" binding:"required,min=1"`
}