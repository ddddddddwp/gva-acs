package service

import (
	"sort"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
)

type DataModelService struct{}

// GetDataModelList 分页查询设备的数据模型
// 支持按名称前缀模糊搜索
func (s *DataModelService) GetDataModelList(deviceID uint, prefix string, offset, limit int) (list []model.DataModelValue, total int64, err error) {
	db := global.GVA_DB.Model(&model.DataModelValue{}).Where("device_id = ?", deviceID)
	if prefix != "" {
		// 仅查询直接子节点（叶子参数），不包含深层嵌套的孙节点
		// 例如 prefix="Device.DeviceInfo."
		// 匹配 "Device.DeviceInfo.SerialNumber"
		// 排除 "Device.DeviceInfo.MemoryStatus.Free" (包含额外的点)
		db = db.Where("name LIKE ? AND name NOT LIKE ?", prefix+"%", prefix+"%.%")
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	err = db.Order("name ASC").Offset(offset).Limit(limit).Find(&list).Error
	return
}

// GetDataModelStructure 获取设备的数据模型结构（去重后的对象路径）
// 例如返回: Device.DeviceInfo., Device.ManagementServer.
func (s *DataModelService) GetDataModelStructure(deviceID uint) (paths []string, err error) {
	var names []string
	// 获取所有参数名（包含对象和叶子）
	// 注意：虽然我们不再存储以 "." 结尾的对象，但为了兼容旧数据和 robust，
	// 我们遍历所有 name 并解析出路径。
	err = global.GVA_DB.Model(&model.DataModelValue{}).
		Where("device_id = ?", deviceID).
		Pluck("name", &names).Error
	if err != nil {
		return
	}

	uniquePaths := make(map[string]struct{})
	for _, name := range names {
		// 解析路径
		// 例如 "Device.Time.CurrentLocalTime" -> "Device.", "Device.Time."
		for i, r := range name {
			if r == '.' {
				path := name[:i+1]
				uniquePaths[path] = struct{}{}
			}
		}
	}

	paths = make([]string, 0, len(uniquePaths))
	for p := range uniquePaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return
}

// GetDataModelValues 查询设备指定路径下的参数值
func (s *DataModelService) GetDataModelValues(deviceID uint, pathPrefix string) (list []model.DataModelValue, err error) {
	// 查询 pathPrefix 下的直接子节点（不包含孙节点，或者根据需求包含所有子孙节点）
	// 这里实现为查询所有以 pathPrefix 开头的叶子节点（即有值的参数）
	db := global.GVA_DB.Model(&model.DataModelValue{}).Where("device_id = ?", deviceID)

	if pathPrefix != "" {
		db = db.Where("name LIKE ? AND name NOT LIKE ?", pathPrefix+"%", pathPrefix+"%.%")
	}

	err = db.Order("name ASC").Find(&list).Error
	return
}
