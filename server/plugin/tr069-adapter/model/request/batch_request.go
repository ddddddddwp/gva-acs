package request

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchOperationRequest 批量操作请求
type BatchOperationRequest struct {
	IDs       []uint                 `json:"ids" binding:"required,min=1"`
	Operation string                 `json:"operation" binding:"required"`
	Data      map[string]interface{} `json:"data"`
}