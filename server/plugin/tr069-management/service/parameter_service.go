package service

import (
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
)

// ParameterService 参数服务
type ParameterService struct{}

// GetParametersByDevice 获取设备参数列表
func (s *ParameterService) GetParametersByDevice(req request.ParameterSearch) ([]model.Parameter, int64, error) {
	var parameters []model.Parameter
	var total int64

	db := global.GVA_DB.Model(&model.Parameter{})

	// 构建查询条件
	if req.DeviceID != 0 {
		db = db.Where("device_id = ?", req.DeviceID)
	}
	if req.Name != "" {
		db = db.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Category != "" {
		db = db.Where("category = ?", req.Category)
	}
	if req.Tags != "" {
		db = db.Where("tags LIKE ?", "%"+req.Tags+"%")
	}

	// 计算总数
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	err = db.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Order("name asc").Find(&parameters).Error
	if err != nil {
		return nil, 0, err
	}

	return parameters, total, nil
}

// GetParameterByID 根据ID获取参数
func (s *ParameterService) GetParameterByID(id uint) (*model.Parameter, error) {
	var parameter model.Parameter
	err := global.GVA_DB.First(&parameter, id).Error
	if err != nil {
		return nil, err
	}
	return &parameter, nil
}

// GetParameterByName 根据名称获取参数
func (s *ParameterService) GetParameterByName(deviceID uint, name string) (*model.Parameter, error) {
	var parameter model.Parameter
	err := global.GVA_DB.Where("device_id = ? AND name = ?", deviceID, name).First(&parameter).Error
	if err != nil {
		return nil, err
	}
	return &parameter, nil
}

// CreateParameter 创建参数
func (s *ParameterService) CreateParameter(parameter model.Parameter) (*model.Parameter, error) {
	// 检查设备是否存在
	var device model.TR069Device
	err := global.GVA_DB.First(&device, parameter.DeviceID).Error
	if err != nil {
		return nil, errors.New("设备不存在")
	}

	// 检查参数是否已存在
	var count int64
	global.GVA_DB.Model(&model.Parameter{}).Where("device_id = ? AND name = ?", parameter.DeviceID, parameter.Name).Count(&count)
	if count > 0 {
		return nil, errors.New("参数已存在")
	}

	// 设置最后修改时间
	parameter.LastChanged = time.Now()

	// 创建参数
	err = global.GVA_DB.Create(&parameter).Error
	if err != nil {
		return nil, err
	}

	return &parameter, nil
}

// UpdateParameter 更新参数
func (s *ParameterService) UpdateParameter(id uint, updates map[string]interface{}) error {
	// 检查参数是否存在
	var parameter model.Parameter
	err := global.GVA_DB.First(&parameter, id).Error
	if err != nil {
		return errors.New("参数不存在")
	}

	// 设置最后修改时间
	updates["last_changed"] = time.Now()

	// 更新参数
	return global.GVA_DB.Model(&parameter).Updates(updates).Error
}

// DeleteParameter 删除参数
func (s *ParameterService) DeleteParameter(id uint) error {
	return global.GVA_DB.Delete(&model.Parameter{}, id).Error
}

// SetDeviceParameters 设置设备参数
func (s *ParameterService) SetDeviceParameters(req request.SetParametersRequest) error {
	// 检查设备是否存在
	var device model.TR069Device
	err := global.GVA_DB.First(&device, req.DeviceID).Error
	if err != nil {
		return errors.New("设备不存在")
	}

	// 遍历参数列表，更新或创建参数
	for _, param := range req.Parameters {
		var parameter model.Parameter
		result := global.GVA_DB.Where("device_id = ? AND name = ?", req.DeviceID, param.Name).First(&parameter)

		if result.Error != nil {
			// 参数不存在，创建新参数
			newParam := model.Parameter{
				DeviceID:     req.DeviceID,
				Name:         param.Name,
				Value:        param.Value,
				Type:         param.Type,
				Writable:     true,
				Notification: 0,
				LastChanged:  time.Now(),
			}
			if err := global.GVA_DB.Create(&newParam).Error; err != nil {
				return err
			}
		} else {
			// 参数存在，更新参数值
			updates := map[string]interface{}{
				"value":        param.Value,
				"last_changed": time.Now(),
			}
			if param.Type != "" {
				updates["type"] = param.Type
			}
			if err := global.GVA_DB.Model(&parameter).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// GetParameterCategories 获取参数分类列表
func (s *ParameterService) GetParameterCategories(deviceID uint) ([]string, error) {
	var categories []string
	err := global.GVA_DB.Model(&model.Parameter{}).
		Where("device_id = ? AND category != ''", deviceID).
		Distinct().
		Pluck("category", &categories).Error
	return categories, err
}