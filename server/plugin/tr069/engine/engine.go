package engine

import (
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/ddddddddwp/tr069-core-only/pkg/core/defaults"
)

type Deps struct {
	Parser       tr069.Parser
	Builder      tr069.Builder
	SessionStore core.SessionStore
	DeviceRepo   core.DeviceRepo
	CommandRepo  core.CommandRepo
	CommandQueue core.CommandSource
	ExecutorReg  core.ExecutorRegistry
	InflightRepo core.InflightRepo
	Hook         core.CorrelationHook
}

func New(deps Deps) (*core.DefaultEngine, error) {
	parser := deps.Parser
	if parser == nil {
		parser = factory.NewParser(tr069.WithStrictMode(false))
	}
	builder := deps.Builder
	if builder == nil {
		builder = factory.NewBuilder()
	}
	store := deps.SessionStore
	if store == nil {
		if adapter.RedisAvailable() {
			store = adapter.NewRedisSessionStore(0)
		} else {
			store = defaults.NewMemorySessionStore(0)
		}
	}
	devRepo := deps.DeviceRepo
	if devRepo == nil {
		devRepo = new(adapter.GormDeviceRepo)
	}
	cmdRepo := deps.CommandRepo
	if cmdRepo == nil {
		if adapter.DBAvailable() {
			cmdRepo = new(adapter.GormCommandRepo)
		} else {
			cmdRepo = defaults.NewMemoryCommandRepo()
		}
	}
	queue := deps.CommandQueue
	if queue == nil {
		if adapter.RedisAvailable() {
			queue = adapter.NewRedisCommandSource(adapter.RedisCommandSourceConfig{})
		} else {
			queue = defaults.NewMemoryQueue()
		}
	}
	reg := deps.ExecutorReg
	if reg == nil {
		r := defaults.NewExecutorRegistry()
		r.Register("GetRPCMethods", &defaults.GetRPCMethodsExecutor{})
		r.Register("GetParameterValues", &defaults.GetParameterValuesExecutor{})
		r.Register("GetParameterNames", &defaults.GetParameterNamesExecutor{DefaultPath: "Device.", DefaultNextLvl: true})
		r.Register("SetParameterValues", &defaults.SetParameterValuesExecutor{})
		r.Register("Reboot", &defaults.RebootExecutor{})
		r.Register("Download", &defaults.DownloadExecutor{})
		r.Register("Upload", &defaults.UploadExecutor{})
		reg = r
	}
	inflight := deps.InflightRepo
	if inflight == nil {
		if adapter.RedisAvailable() {
			inflight = adapter.NewRedisInflightRepo(10 * time.Minute)
		} else {
			inflight = adapter.NewMemoryInflightRepo(10 * time.Minute)
		}
	}
	hook := deps.Hook
	if hook == nil {
		base := defaults.NewInflightCorrelationHook(inflight, cmdRepo)
		var ingest core.CommandIngest
		if adapter.RedisAvailable() {
			ingest = adapter.NewRedisCommandIngest("")
		} else {
			if mq, ok := queue.(*defaults.MemoryQueue); ok {
				ingest = defaults.NewMemoryCommandIngest(mq)
			}
		}
		hook = adapter.NewDataModelHook(base, inflight, ingest)
	}
	return core.NewEngine(core.Config{
		Parser:          parser,
		Builder:         builder,
		SessionStore:    store,
		DeviceRepo:      devRepo,
		CommandRepo:     cmdRepo,
		ExecutorReg:     reg,
		CommandSource:   queue,
		CorrelationHook: hook,
	})
}

var (
	once sync.Once
	inst *core.DefaultEngine
	err  error
)

func Get() (*core.DefaultEngine, error) {
	once.Do(func() {
		inst, err = New(Deps{})
	})
	return inst, err
}
