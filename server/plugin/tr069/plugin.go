package tr069

import (
	"context"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/initialize"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/router"
	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var Plugin = new(tr069Plugin)

var registerLifecycle sync.Once

type tr069Plugin struct{}

func (p *tr069Plugin) Register(group *gin.Engine) {
	registerLifecycle.Do(func() {
		utils.GlobalSystemEvents.RegisterConfigChangeHandler(initialize.ReloadConfig)
		utils.GlobalSystemEvents.RegisterShutdownHandler(func(ctx context.Context) error {
			adapter.StopRedisDispatcher()
			return adapter.StopCommandWakeConsumer(ctx)
		})
	})
	initialize.ReloadConfig()
	tr069Global.SetStartupConfig(config.CurrentRuntime().Settings)
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
	if err := adapter.StartCommandWakeConsumer(context.Background(), adapter.CommandWakeConsumerConfig{
		BlockTimeout: 2 * time.Second,
		ConnectionRequest: adapter.ConnectionRequestConfig{
			Timeout: 5 * time.Second,
			Retries: 1,
		},
	}); err != nil {
		global.GVA_LOG.Error("failed to start TR-069 command wake consumer", zap.Error(err))
	}

	r := group.Group("tr069")
	r.Use(middleware.JWTAuth()).Use(middleware.CasbinHandler())
	deviceRouter := new(router.DeviceRouter)
	deviceRouter.InitDeviceRouter(r)
	alarmRouter := new(router.AlarmRouter)
	alarmRouter.InitAlarmRouter(r)
}

func (p *tr069Plugin) RouterPath() string {
	return "tr069"
}
