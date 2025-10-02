package tr069server

import (
	"context"
	"encoding/xml"
	"fmt"

	tr069core "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"go.uber.org/zap"
)

// SOAPHandler SOAP消息处理器
type SOAPHandler struct {
	logger       *zap.Logger
	tr069Builder tr069core.Builder
}

// NewSOAPHandler 创建SOAP处理器
func NewSOAPHandler(logger *zap.Logger) *SOAPHandler {
	return &SOAPHandler{
		logger: logger,
	}
}

// processTR069CoreInform 使用TR069-Core处理Inform消息
func (h *SOAPHandler) processTR069CoreInform(deviceInfo *DeviceInfo, inform *Inform) (interface{}, error) {
	// 将TR069-Adapter的Inform消息转换为TR069-Core的Message格式
	ctx := context.Background()

	// 构造TR069-Core的Message对象
	message := &tr069core.Message{
		Method: "Inform",
		ID:     inform.DeviceId.OUI,
		Params: map[string]interface{}{
			"DeviceId": map[string]string{
				"Manufacturer": inform.DeviceId.Manufacturer,
				"OUI":          inform.DeviceId.OUI,
				"ProductClass": inform.DeviceId.ProductClass,
				"SerialNumber": inform.DeviceId.SerialNumber,
			},
			"Event":        inform.Event,
			"MaxEnvelopes": inform.MaxEnvelopes,
			"CurrentTime":  inform.CurrentTime,
			"RetryCount":   inform.RetryCount,
		},
	}

	// 转换参数列表
	params := make([]tr069core.Parameter, 0, len(inform.ParameterList))
	for _, param := range inform.ParameterList {
		params = append(params, tr069core.Parameter{
			Name:  param.Name,
			Value: param.Value,
			Type:  param.Type,
		})
	}
	message.Params["ParameterList"] = params

	// 使用TR069-Core解析消息
	response, err := h.tr069Builder.BuildRPCResponse(ctx, "InformResponse", map[string]interface{}{
		"MaxEnvelopes": 1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to build inform response: %w", err)
	}

	return response, nil
}

// buildEmptyResponse 构建空响应
func (h *SOAPHandler) buildEmptyResponse() ([]byte, error) {
	response := &SOAPEnvelope{
		XMLName: xml.Name{Local: "soap:Envelope"},
		Header:  &SOAPHeader{},
		Body:    &SOAPBody{},
	}

	return xml.Marshal(response)
}
