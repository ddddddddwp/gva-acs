package api

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/gin-gonic/gin"
)

type DeviceApi struct{}

// GetDeviceList
// @Tags TR069
// @Summary Get device list
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=object,msg=string} "Get device list"
// @Router /tr069/device/list [get]
func (a *DeviceApi) GetDeviceList(c *gin.Context) {
	var devices []model.Device
	err := global.GVA_DB.Find(&devices).Error
	if err != nil {
		response.FailWithMessage("Failed to get device list", c)
		return
	}
	response.OkWithData(devices, c)
}
