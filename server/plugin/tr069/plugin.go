package tr069

import (
	"context"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/initialize"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/router"
	"github.com/gin-gonic/gin"
)

var Plugin = new(tr069Plugin)

type tr069Plugin struct{}

func (p *tr069Plugin) Register(group *gin.Engine) {
	initialize.Viper()
	initialize.Gorm(context.Background())
	initialize.Api(context.Background())
	initialize.Menu(context.Background())
	initialize.StartTR069Server()

	adapter.StartRedisDispatcher(context.Background(), adapter.RedisDispatcherConfig{
		IngestStream: adapter.RedisIngestStreamKey,
		Group:        adapter.RedisDispatcherGroup,
		Block:        2 * time.Second,
		BatchSize:    128,
		ClaimIdle:    60 * time.Second,
	})

	r := group.Group("tr069")
	deviceRouter := new(router.DeviceRouter)
	deviceRouter.InitDeviceRouter(r)
	alarmRouter := new(router.AlarmRouter)
	alarmRouter.InitAlarmRouter(r)
}

func (p *tr069Plugin) RouterPath() string {
	return "tr069"
}
