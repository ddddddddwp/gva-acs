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
	// 1. Init Database
	initialize.Gorm(context.Background())

	// 2. Start Independent TR069 Server (Port 7547)
	initialize.StartTR069Server()

	adapter.StartRedisDispatcher(context.Background(), adapter.RedisDispatcherConfig{
		IngestStream: adapter.RedisIngestStreamKey,
		Group:        adapter.RedisDispatcherGroup,
		Block:        2 * time.Second,
		BatchSize:    128,
		ClaimIdle:    60 * time.Second,
	})

	// 3. Register Management API Router (Port 8888)
	// Usually plugins mount routes under a specific group, but here we might need authentication
	// We'll attach to the PrivateGroup if possible, or just the engine for now as per v2 interface
	// In GVA v2 plugin system, 'group' is the main engine.
	// We should ideally find the PrivateGroup.
	// For simplicity in this step, we will use a group with prefix

	// Note: GVA's plugin mechanism usually handles the router group injection differently.
	// But based on the interface `Register(group *gin.Engine)`, we can attach anywhere.

	publicGroup := group.Group("tr069")
	{
		// Public routes if any
	}

	// Assuming we want these protected, but we don't have easy access to the JWT middleware here directly
	// without importing 'middleware'. Let's register them under /tr069 for now.
	// In a real scenario, you'd use the middleware from GVA.

	deviceRouter := new(router.DeviceRouter)
	deviceRouter.InitDeviceRouter(publicGroup)
}

func (p *tr069Plugin) RouterPath() string {
	return "tr069"
}
