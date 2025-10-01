package api

// ParseRequest 解析请求结构
type ParseRequest struct {
	XMLData string `json:"xml_data"`
}

// ParseResponse 解析响应结构
type ParseResponse struct {
	Success   bool        `json:"success"`
	Message   *Message    `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	ParseTime string      `json:"parse_time"`
}

// BuildRequest 构建请求结构
type BuildRequest struct {
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
}

// BuildResponse 构建响应结构
type BuildResponse struct {
	Success   bool   `json:"success"`
	XMLData   string `json:"xml_data,omitempty"`
	Error     string `json:"error,omitempty"`
	BuildTime string `json:"build_time"`
}

// Message TR069消息结构（简化版，用于调试显示）
type Message struct {
	Method     string                 `json:"method"`
	SessionID  string                 `json:"session_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

// StatusResponse 状态响应结构
type StatusResponse struct {
	LibraryVersion string            `json:"library_version"`
	Status         string            `json:"status"`
	Statistics     map[string]int64  `json:"statistics"`
	Configuration  map[string]string `json:"configuration"`
}