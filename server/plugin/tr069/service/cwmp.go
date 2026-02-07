package service

import (
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CWMPService struct{}

func (s *CWMPService) HandleMessage(msg *tr069.Message, clientIP string) (*tr069.Message, error) {
	// Simple logic: Update device info on 'Inform'
	if msg.Method == "Inform" {
		var device model.Device
		err := global.GVA_DB.Where("serial_number = ?", msg.DeviceID.SerialNumber).First(&device).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果是除了 RecordNotFound 以外的错误，记录日志并返回
			global.GVA_LOG.Error("Query device failed", zap.Error(err))
			return nil, err
		}
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			device = model.Device{
				SerialNumber: msg.DeviceID.SerialNumber,
				IP:           clientIP,
				LastOnline:   time.Now(),
			}
			if err := global.GVA_DB.Create(&device).Error; err != nil {
				global.GVA_LOG.Error("Create device failed", zap.Error(err))
				return nil, err
			}
		} else {
			device.IP = clientIP
			device.LastOnline = time.Now()
			if err := global.GVA_DB.Save(&device).Error; err != nil {
				global.GVA_LOG.Error("Update device failed", zap.Error(err))
				return nil, err
			}
		}
	}

	// Mock response logic using the real SDK struct
	// In a real scenario, you'd construct a proper InformResponse
	resp := &tr069.Message{
		ID:     msg.ID,
		Method: msg.Method + "Response", // e.g. InformResponse
		// Populate other fields as needed
	}
	return resp, nil
}
