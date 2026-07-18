package request

type ArtifactListRequest struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	DeviceID    uint   `form:"deviceId"`
	Channel     string `form:"channel"`
	Status      string `form:"status"`
	CreatedFrom string `form:"createdFrom"`
	CreatedTo   string `form:"createdTo"`
}
