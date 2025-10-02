package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
)

type SessionService struct {
	bridgeService *TR069BridgeService
}

// GetSessionList 获取会话列表
func (s *SessionService) GetSessionList(req request.SessionSearch) ([]interface{}, int64, error) {
	// 这里可以调用TR069-Core的会话列表接口，暂时返回空
	return []interface{}{}, 0, nil
}

// GetSessionByID 根据ID获取会话详情
func (s *SessionService) GetSessionByID(id uint) (interface{}, error) {
	// 这里可以调用TR069-Core的会话详情接口，暂时返回空
	return nil, nil
}

// GetDeviceSessions 获取设备会话列表
func (s *SessionService) GetDeviceSessions(deviceID uint) ([]interface{}, error) {
	return s.bridgeService.GetDeviceSessions(deviceID)
}