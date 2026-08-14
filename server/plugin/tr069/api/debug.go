package api

import (
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/gin-gonic/gin"
)

type DebugApi struct{}

func (a *DebugApi) GetTrace(c *gin.Context) {
	traceID := c.Param("traceId")
	if traceID == "" {
		response.FailWithMessage("traceId 不能为空", c)
		return
	}
	entries := trace.Get(c.Request.Context(), traceID)
	response.OkWithDetailed(entries, "获取成功", c)
}
