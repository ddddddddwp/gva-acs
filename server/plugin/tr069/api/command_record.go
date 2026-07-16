package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	commandResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/gin-gonic/gin"
)

const commandRecordTimeLayout = "2006-01-02 15:04:05"

type CommandRecordApi struct{}

var authorizeCommandRetry = hasCommandRetryOperationPermission

func hasCommandRetryOperationPermission(c *gin.Context, command model.Command) bool {
	if command.DeviceID == 0 || command.Operation == "" {
		return false
	}
	authorityID := utils.GetUserAuthorityId(c)
	if authorityID == 0 {
		return false
	}
	routeOperation := strings.ToLower(command.Operation[:1]) + command.Operation[1:]
	path := fmt.Sprintf("/tr069/command/%d/%s", command.DeviceID, routeOperation)
	allowed, err := utils.GetCasbin().Enforce(strconv.Itoa(int(authorityID)), path, "POST")
	return err == nil && allowed
}

func parseCommandRecordTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation(commandRecordTimeLayout, value, time.Local)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (a *CommandRecordApi) List(c *gin.Context) {
	var in req.CommandRecordListRequest
	if err := c.ShouldBindQuery(&in); err != nil {
		response.FailWithMessage("查询参数错误", c)
		return
	}
	createdFrom, err := parseCommandRecordTime(in.CreatedFrom)
	if err != nil {
		response.FailWithMessage("开始时间格式错误", c)
		return
	}
	createdTo, err := parseCommandRecordTime(in.CreatedTo)
	if err != nil {
		response.FailWithMessage("结束时间格式错误", c)
		return
	}
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 10
	} else if in.PageSize > 100 {
		in.PageSize = 100
	}
	commands, total, err := service.NewCommandStore(global.GVA_DB).List(c.Request.Context(), service.CommandListFilter{
		DeviceID: in.DeviceID, DeviceSerial: strings.TrimSpace(in.DeviceSerial),
		Operation: in.Operation, Status: in.Status, CommandID: strings.TrimSpace(in.CommandID),
		CreatedFrom: createdFrom, CreatedTo: createdTo,
		Offset: (in.Page - 1) * in.PageSize, Limit: in.PageSize,
	})
	if err != nil {
		response.FailWithMessage("获取 RPC 记录失败", c)
		return
	}
	list := make([]commandResponse.CommandRecordSummary, 0, len(commands))
	for _, command := range commands {
		list = append(list, commandResponse.NewCommandRecordSummary(command))
	}
	response.OkWithDetailed(response.PageResult{List: list, Total: total, Page: in.Page, PageSize: in.PageSize}, "获取成功", c)
}

func (a *CommandRecordApi) Detail(c *gin.Context) {
	commandID := strings.TrimSpace(c.Param("commandId"))
	if commandID == "" {
		response.FailWithMessage("Command ID 不能为空", c)
		return
	}
	detail, err := service.NewCommandStore(global.GVA_DB).Detail(c.Request.Context(), commandID)
	if err != nil {
		response.FailWithMessage("获取 RPC 记录详情失败", c)
		return
	}
	response.OkWithDetailed(commandResponse.CommandRecordDetailFrom(detail.Command, detail.Events, detail.XML), "获取成功", c)
}

func (a *CommandRecordApi) Retry(c *gin.Context) {
	commandID := strings.TrimSpace(c.Param("commandId"))
	if commandID == "" {
		response.FailWithMessage("Command ID 不能为空", c)
		return
	}
	detail, err := service.NewCommandStore(global.GVA_DB).Detail(c.Request.Context(), commandID)
	if err != nil {
		response.FailWithMessage("获取原 RPC 记录失败", c)
		return
	}
	if !authorizeCommandRetry(c, detail.Command) {
		response.FailWithMessage("缺少原操作的下发权限", c)
		return
	}
	result, err := commandService.Retry(c.Request.Context(), commandID)
	if err != nil {
		response.FailWithMessage("重新下发失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "命令已重新提交", c)
}
