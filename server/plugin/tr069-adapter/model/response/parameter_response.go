package response

import (
	"time"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
)

// ParameterResponse 参数响应
type ParameterResponse struct {
	ID           uint      `json:"id"`
	DeviceID     uint      `json:"deviceId"`
	Name         string    `json:"name"`
	Value        string    `json:"value"`
	Type         string    `json:"type"`
	Writable     bool      `json:"writable"`
	Notification int       `json:"notification"`
	LastChanged  time.Time `json:"lastChanged"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ParameterListResponse 参数列表响应
type ParameterListResponse struct {
	List     []ParameterResponse `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

// ParameterTreeNode 参数树节点
type ParameterTreeNode struct {
	Name         string               `json:"name"`
	FullPath     string               `json:"fullPath"`
	Value        string               `json:"value,omitempty"`
	Type         string               `json:"type,omitempty"`
	Writable     bool                 `json:"writable"`
	Notification int                  `json:"notification"`
	IsLeaf       bool                 `json:"isLeaf"`
	Children     []*ParameterTreeNode `json:"children,omitempty"`
}

// ParameterTreeResponse 参数树响应
type ParameterTreeResponse struct {
	DeviceID uint                 `json:"deviceId"`
	Root     *ParameterTreeNode   `json:"root"`
}

// GetParameterValuesResponse 获取参数值响应
type GetParameterValuesResponse struct {
	DeviceID   uint                     `json:"deviceId"`
	Parameters []model.ParameterValue   `json:"parameters"`
	Status     string                   `json:"status"`
	Message    string                   `json:"message"`
}

// SetParameterValuesResponse 设置参数值响应
type SetParameterValuesResponse struct {
	DeviceID  uint   `json:"deviceId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// GetParameterNamesResponse 获取参数名称响应
type GetParameterNamesResponse struct {
	DeviceID   uint                    `json:"deviceId"`
	Parameters []model.ParameterInfo   `json:"parameters"`
	Status     string                  `json:"status"`
	Message    string                  `json:"message"`
}

// GetParameterAttributesResponse 获取参数属性响应
type GetParameterAttributesResponse struct {
	DeviceID   uint                        `json:"deviceId"`
	Parameters []model.ParameterAttribute  `json:"parameters"`
	Status     string                      `json:"status"`
	Message    string                      `json:"message"`
}

// SetParameterAttributesResponse 设置参数属性响应
type SetParameterAttributesResponse struct {
	DeviceID  uint   `json:"deviceId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// AddObjectResponse 添加对象响应
type AddObjectResponse struct {
	DeviceID     uint   `json:"deviceId"`
	ObjectName   string `json:"objectName"`
	InstanceNumber int  `json:"instanceNumber"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	RequestID    string `json:"requestId,omitempty"`
}

// DeleteObjectResponse 删除对象响应
type DeleteObjectResponse struct {
	DeviceID   uint   `json:"deviceId"`
	ObjectName string `json:"objectName"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	RequestID  string `json:"requestId,omitempty"`
}