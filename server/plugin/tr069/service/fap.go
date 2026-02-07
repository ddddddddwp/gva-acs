package service

import (
	"errors"
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
)

type FAPService struct{}

// GetFAPInfo returns the FAP configuration for a given device
func (s *FAPService) GetFAPInfo(deviceID uint) (*model.FAPService, error) {
	var fap model.FAPService
	err := global.GVA_DB.Where("device_id = ?", deviceID).First(&fap).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // Not found is not an error, just empty
	}
	return &fap, err
}

// SyncFAPInfo creates a task to fetch TR-196 parameters from the device
// Note: This function assumes there is a task queue mechanism (which we will implement later)
// For now, it just logs the intent.
func (s *FAPService) SyncFAPInfo(deviceID uint) error {
	// TODO: Create a "GetParameterValues" task in the DB/Redis queue
	// Parameters to fetch:
	// Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID
	// Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL
	// Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.TxPower
	
	global.GVA_LOG.Info("Creating Sync Task for Device", 
		// zap.Uint("deviceID", deviceID),
	)
	return nil
}

// UpdateFAPConfig creates a task to push new configuration to the device
func (s *FAPService) UpdateFAPConfig(deviceID uint, config model.FAPService) error {
	// TODO: Create a "SetParameterValues" task in the DB/Redis queue
	
	global.GVA_LOG.Info("Creating Configure Task for Device",
		// zap.Uint("deviceID", deviceID),
	)
	return nil
}
