package request

type LogCollectionRequest struct {
	FileType     string `json:"fileType" binding:"required"`
	DelaySeconds int    `json:"delaySeconds" binding:"min=0"`
}

type ArtifactListRequest struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	SerialNumber string `form:"serialNumber"`
	CreatedFrom  string `form:"createdFrom"`
	CreatedTo    string `form:"createdTo"`
}
