package router

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/api"
	"github.com/gin-gonic/gin"
)

type AlarmRouter struct{}

func (r *AlarmRouter) InitAlarmRouter(Router *gin.RouterGroup) {
	alarmRouter := Router.Group("alarm")
	alarmApi := new(api.AlarmApi)
	{
		alarmRouter.GET("list", alarmApi.GetAlarmList)
	}
}
