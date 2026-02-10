package request

type GetParameterValuesRequest struct {
	Paths []string `json:"paths" form:"paths"`
}

type SetParameterValueItem struct {
	Name  string      `json:"name" form:"name"`
	Value interface{} `json:"value" form:"value"`
	Type  string      `json:"type" form:"type"`
}

type SetParameterValuesRequest struct {
	ParameterKey string               `json:"parameterKey" form:"parameterKey"`
	Parameters   []SetParameterValueItem `json:"parameters" form:"parameters"`
}

