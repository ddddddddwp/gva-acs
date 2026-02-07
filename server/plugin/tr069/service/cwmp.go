package service

import (
	"errors"
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/lib/parser"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
	"time"
)

type CWMPService struct{}

func (s *CWMPService) HandleMessage(msg *parser.Message, clientIP string) (*parser.Message, error) {
	// Simple logic: Update device info on 'Inform'
	if msg.Method == "Inform" {
		var device model.Device
		err := global.GVA_DB.Where("serial_number = ?", msg.DeviceID.SerialNumber).First(&device).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			device = model.Device{
				SerialNumber: msg.DeviceID.SerialNumber,
				IP:           clientIP,
				LastOnline:   time.Now(),
			}
			global.GVA_DB.Create(&device)
		} else {
			device.IP = clientIP
			device.LastOnline = time.Now()
			global.GVA_DB.Save(&device)
		}
	}
	
	// For now, just return the same message as a placeholder response
	// In reality, this would determine the next method to call on the device
	return msg, nil
}
