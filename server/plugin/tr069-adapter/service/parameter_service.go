package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
)

type ParameterService struct {
	bridgeService *TR069BridgeService
}

// GetParametersByDevice 获取设备参数
func (p *ParameterService) GetParametersByDevice(deviceID uint, parameterNames []string) ([]interface{}, error) {
	return p.bridgeService.GetParametersByDevice(deviceID, parameterNames)
}

// SetDeviceParameters 设置设备参数
func (p *ParameterService) SetDeviceParameters(deviceID uint, parameters []request.ParameterValuePair) error {
	req := request.SetParametersRequest{
		DeviceID:   deviceID,
		Parameters: parameters,
	}
	return p.bridgeService.SetDeviceParameters(deviceID, req)
}

// GetParameterHistory 获取参数历史记录
func (p *ParameterService) GetParameterHistory(deviceID uint, parameterName string) ([]interface{}, error) {
	// 这里可以调用TR069-Core的参数历史接口，暂时返回空
	return []interface{}{}, nil
}