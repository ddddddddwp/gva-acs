package request

type GetParameterValuesRequest struct {
	Paths []string `json:"paths" binding:"required,min=1,dive,required"`
}

type GetParameterNamesRequest struct {
	ParameterPath string `json:"parameterPath" binding:"required"`
	NextLevel     bool   `json:"nextLevel"`
}

type GetParameterAttributesRequest struct {
	ParameterNames []string `json:"parameterNames" binding:"required,min=1,dive,required"`
}

type SetParameterValue struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type SetParameterValuesRequest struct {
	ParameterKey string              `json:"parameterKey"`
	Parameters   []SetParameterValue `json:"parameters" binding:"required,min=1"`
}

type SetParameterAttribute struct {
	Name               string   `json:"name"`
	NotificationChange bool     `json:"notificationChange"`
	Notification       int      `json:"notification"`
	AccessListChange   bool     `json:"accessListChange"`
	AccessList         []string `json:"accessList"`
}

type SetParameterAttributesRequest struct {
	ParameterAttributes []SetParameterAttribute `json:"parameterAttributes" binding:"required,min=1"`
}

type ObjectRequest struct {
	ObjectName   string `json:"objectName" binding:"required"`
	ParameterKey string `json:"parameterKey"`
}

type DownloadRequest struct {
	FileType       string `json:"fileType"`
	URL            string `json:"url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	FileSize       int    `json:"fileSize"`
	TargetFileName string `json:"targetFileName"`
	DelaySeconds   int    `json:"delaySeconds"`
	SuccessURL     string `json:"successURL"`
	FailureURL     string `json:"failureURL"`
}

type UploadRequest struct {
	FileType     string `json:"fileType"`
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DelaySeconds int    `json:"delaySeconds"`
}
