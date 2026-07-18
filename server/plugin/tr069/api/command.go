package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type CommandApi struct{}

var commandPayloadProtector = adapter.NewCompositeCommandPayloadCodec(
	adapter.NewConnectionProfilePayloadProtector(
		adapter.NewConnectionProfileRepository(nil, adapter.NewRuntimeCredentialCipher()),
	),
	adapter.LogUploadPayloadCodec{},
)

var commandService = service.NewCommandService(service.NewCommandManager(
	nil,
	adapter.EnqueueImmediate,
	service.WithCommandPayloadProtector(commandPayloadProtector),
	service.WithCommandCreatedHook(service.NewActiveUploadTaskHook(nil)),
))

func commandFailureMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrDeviceOffline):
		return "设备离线，无法下发任务"
	case errors.Is(err, service.ErrCommandQueueUnavailable):
		return "命令队列不可用"
	default:
		return "任务下发失败: " + err.Error()
	}
}

func commandSubmitMessage(result service.SubmitResult) string {
	if result.Status == model.CommandStatusFailed {
		return "任务已持久化，但设备唤醒调度失败"
	}
	return "任务已创建并进入持久化队列"
}

func submitCommand(c *gin.Context, operation string, request any) {
	deviceID, err := strconv.ParseUint(c.Param("deviceId"), 10, 64)
	if err != nil || deviceID == 0 {
		response.FailWithMessage("设备ID错误", c)
		return
	}
	result, err := commandService.Submit(c.Request.Context(), uint(deviceID), operation, request)
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(result, commandSubmitMessage(result), c)
}

func bindAndSubmitCommand[T any](c *gin.Context, operation string) {
	var in T
	if err := c.ShouldBindJSON(&in); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	submitCommand(c, operation, in)
}

// SyncRPCMethods
// @Tags TR069
// @Summary 下发 GetRPCMethods
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/getRPCMethods [post]
func (a *CommandApi) SyncRPCMethods(c *gin.Context) {
	submitCommand(c, "GetRPCMethods", nil)
}

// GetParameterValues
// @Tags TR069
// @Summary 下发 GetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body request.GetParameterValuesRequest true "查询参数列表"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/getParameterValues [post]
func (a *CommandApi) GetParameterValues(c *gin.Context) {
	bindAndSubmitCommand[req.GetParameterValuesRequest](c, "GetParameterValues")
}

// GetParameterNames 下发 GetParameterNames。
func (a *CommandApi) GetParameterNames(c *gin.Context) {
	bindAndSubmitCommand[req.GetParameterNamesRequest](c, "GetParameterNames")
}

// GetParameterAttributes 下发 GetParameterAttributes。
func (a *CommandApi) GetParameterAttributes(c *gin.Context) {
	bindAndSubmitCommand[req.GetParameterAttributesRequest](c, "GetParameterAttributes")
}

// SetParameterValues
// @Tags TR069
// @Summary 下发 SetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body request.SetParameterValuesRequest true "设置参数列表"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/setParameterValues [post]
func (a *CommandApi) SetParameterValues(c *gin.Context) {
	bindAndSubmitCommand[req.SetParameterValuesRequest](c, "SetParameterValues")
}

// SetParameterAttributes 下发 SetParameterAttributes。
func (a *CommandApi) SetParameterAttributes(c *gin.Context) {
	bindAndSubmitCommand[req.SetParameterAttributesRequest](c, "SetParameterAttributes")
}

// AddObject 下发 AddObject。
func (a *CommandApi) AddObject(c *gin.Context) {
	bindAndSubmitCommand[req.ObjectRequest](c, "AddObject")
}

// DeleteObject 下发 CWMP DeleteObject；它不会删除 GVA 中的设备记录。
func (a *CommandApi) DeleteObject(c *gin.Context) {
	bindAndSubmitCommand[req.ObjectRequest](c, "DeleteObject")
}

// Download 下发 Download。
func (a *CommandApi) Download(c *gin.Context) {
	bindAndSubmitCommand[req.DownloadRequest](c, "Download")
}

// Upload 下发 Upload。
func (a *CommandApi) Upload(c *gin.Context) {
	var in req.LogCollectionRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	runtime := config.CurrentRuntime().Settings.FileIngress
	logChannel, ok := runtime.Channels["log"]
	if !runtime.Enabled || !ok || !logChannel.Enabled || strings.TrimSpace(runtime.PublicBaseURL) == "" ||
		runtime.Authentication.Username == "" || runtime.Authentication.Password == "" {
		response.FailWithMessage("LOG 文件入口未启用或配置不完整", c)
		return
	}
	submitCommand(c, "Upload", req.UploadRequest{
		FileType: in.FileType, DelaySeconds: in.DelaySeconds,
		URL:      strings.TrimRight(runtime.PublicBaseURL, "/") + logChannel.Path,
		Username: runtime.Authentication.Username, Password: runtime.Authentication.Password,
	})
}

// Reboot 下发 Reboot。
func (a *CommandApi) Reboot(c *gin.Context) {
	submitCommand(c, "Reboot", nil)
}

// FactoryReset 下发 FactoryReset。
func (a *CommandApi) FactoryReset(c *gin.Context) {
	submitCommand(c, "FactoryReset", nil)
}
