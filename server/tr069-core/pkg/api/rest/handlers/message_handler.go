package handlers

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/api/rest"
)

// MessageHandler 处理TR069消息转换的API请求
type MessageHandler struct {
	parser  interfaces.Parser
	builder interfaces.Builder
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(parser interfaces.Parser, builder interfaces.Builder) *MessageHandler {
	return &MessageHandler{
		parser:  parser,
		builder: builder,
	}
}

// RegisterRoutes 注册API路由
func (h *MessageHandler) RegisterRoutes(router interfaces.Router) {
	router.Handle(http.MethodPost, "/messages/parse", h.ParseMessage)
	router.Handle(http.MethodPost, "/messages/build", h.BuildMessage)
	router.Handle(http.MethodPost, "/messages/fault", h.BuildFault)
}

// SetMiddleware 设置中间件
func (h *MessageHandler) SetMiddleware(middleware ...interfaces.Middleware) {
	// 这里可以存储特定于处理器的中间件
}

// ParseMessage 解析TR069 XML消息为JSON
func (h *MessageHandler) ParseMessage(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Failed to read request body: "+err.Error(), 400)
		return
	}

	// 使用解析器解析消息
	message, err := h.parser.ParseMessage(r.Context(), body)
	if err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Failed to parse message: "+err.Error(), 400)
		return
	}

	// 返回解析后的消息
	rest.SendSuccessResponse(w, message)
}

// BuildMessage 从JSON构建TR069 XML消息
func (h *MessageHandler) BuildMessage(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var messageRequest interfaces.Message
	if err := json.NewDecoder(r.Body).Decode(&messageRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}

	// 使用构建器构建消息
	xmlBytes, err := h.builder.BuildMessage(r.Context(), &messageRequest)
	if err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Failed to build message: "+err.Error(), 400)
		return
	}

	// 返回构建的XML消息
	response := map[string]interface{}{
		"xml": string(xmlBytes),
	}
	rest.SendSuccessResponse(w, response)
}

// BuildFault 构建TR069故障消息
func (h *MessageHandler) BuildFault(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var faultRequest struct {
		FaultCode    int    `json:"faultCode"`
		FaultString  string `json:"faultString"`
	}
	if err := json.NewDecoder(r.Body).Decode(&faultRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}

	// 使用构建器构建故障消息
	xmlBytes, err := h.builder.BuildFault(r.Context(), faultRequest.FaultCode, faultRequest.FaultString)
	if err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Failed to build fault message: "+err.Error(), 400)
		return
	}

	// 返回构建的XML故障消息
	response := map[string]interface{}{
		"xml": string(xmlBytes),
	}
	rest.SendSuccessResponse(w, response)
}

// XMLToJSON 将XML转换为JSON
func XMLToJSON(xmlData []byte) ([]byte, error) {
	var data interface{}
	if err := xml.Unmarshal(xmlData, &data); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// JSONToXML 将JSON转换为XML
func JSONToXML(jsonData []byte) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}
	return xml.Marshal(data)
}