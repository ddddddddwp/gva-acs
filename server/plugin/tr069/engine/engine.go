package engine

import (
	"context"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/observability"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/ddddddddwp/tr069-core-only/pkg/core/defaults"
	"go.uber.org/zap"
)

type Deps struct {
	Parser         tr069.Parser
	Builder        tr069.Builder
	SessionStore   core.SessionStore
	DeviceRepo     core.DeviceRepo
	CommandRepo    core.CommandRepo
	CommandQueue   core.CommandSource
	ExecutorReg    core.ExecutorRegistry
	InflightRepo   core.InflightRepo
	Hook           core.CorrelationHook
	EventSink      observability.EventSink
	RuntimeContext context.Context
	CommandWakeup  service.CommandWakeupFunc
}

func New(deps Deps) (*core.DefaultEngine, error) {
	engine, _, err := newEngine(deps)
	return engine, err
}

func newEngine(deps Deps) (*core.DefaultEngine, <-chan struct{}, error) {
	var profileRepository *adapter.ConnectionProfileRepository
	var payloadProtector *adapter.ConnectionProfilePayloadProtector
	var provisioner *adapter.ConnectionCredentialProvisioner
	if adapter.DBAvailable() {
		profileRepository = adapter.NewConnectionProfileRepository(nil, adapter.NewRuntimeCredentialCipher())
		payloadProtector = adapter.NewConnectionProfilePayloadProtector(profileRepository)
		wakeup := deps.CommandWakeup
		if wakeup == nil {
			wakeup = adapter.EnqueueImmediate
		}
		manager := service.NewCommandManager(nil, wakeup, service.WithCommandPayloadProtector(payloadProtector))
		provisioner = adapter.NewConnectionCredentialProvisioner(manager, profileRepository)
	}

	eventSink := deps.EventSink
	if eventSink == nil && adapter.DBAvailable() {
		eventSink = adapter.NewCommandXMLSink(service.NewCommandStore(global.GVA_DB), nil)
	}
	parser := deps.Parser
	if parser == nil {
		options := []tr069.Option{tr069.WithStrictMode(false)}
		if eventSink != nil {
			options = append(options, tr069.WithEventSink(eventSink))
		}
		parser = factory.NewParser(options...)
	}
	builder := deps.Builder
	if builder == nil {
		if eventSink != nil {
			builder = factory.NewBuilder(tr069.WithEventSink(eventSink))
		} else {
			builder = factory.NewBuilder()
		}
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
		devRepo = adapter.NewGormDeviceRepo(nil, profileRepository, provisioner)
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
			cfg := config.CurrentRuntime().Settings
			queueCfg := adapter.RedisCommandSourceConfig{
				LockTTL:              time.Duration(cfg.CommandQueueLockTTL) * time.Second,
				DedupTTL:             time.Duration(cfg.CommandQueueDedupTTL) * time.Second,
				MaxScan:              cfg.CommandQueueMaxScan,
				MaxPendingPerSession: cfg.CommandQueueMaxPendingPerSession,
			}
			var err error
			queue, err = adapter.NewRedisCommandSource(
				queueCfg,
				adapter.WithRedisCommandHydrator(payloadProtector),
			)
			if err != nil {
				global.GVA_LOG.Error("failed to create Redis command source", zap.Error(err))
				queue = defaults.NewMemoryQueue()
			}
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
			ing, err := adapter.NewRedisCommandIngest("")
			if err != nil {
				global.GVA_LOG.Error("failed to create Redis command ingest", zap.Error(err))
			} else {
				ingest = ing
			}
		} else {
			if mq, ok := queue.(*defaults.MemoryQueue); ok {
				ingest = defaults.NewMemoryCommandIngest(mq)
			}
		}
		hookOptions := make([]adapter.DataModelHookOption, 0, 2)
		if profileRepository != nil {
			hookOptions = append(hookOptions, adapter.WithDataModelHookProfiles(profileRepository))
		}
		if provisioner != nil {
			hookOptions = append(hookOptions, adapter.WithDataModelHookProvisioner(provisioner))
		}
		hook = adapter.NewDataModelHook(base, inflight, ingest, hookOptions...)
	}
	engine, err := core.NewEngine(core.Config{
		Parser:          parser,
		Builder:         builder,
		SessionStore:    store,
		DeviceRepo:      devRepo,
		CommandRepo:     cmdRepo,
		ExecutorReg:     reg,
		CommandSource:   queue,
		CorrelationHook: hook,
	})
	if err != nil {
		return nil, nil, err
	}
	var runtimeDone chan struct{}
	if provisioner != nil && deps.RuntimeContext != nil {
		runtimeDone = make(chan struct{})
		go func() {
			defer close(runtimeDone)
			provisioner.Run(deps.RuntimeContext)
		}()
	}
	return engine, runtimeDone, nil
}

var (
	once             sync.Once
	inst             *core.DefaultEngine
	err              error
	runtimeLifecycle struct {
		sync.Mutex
		cancel context.CancelFunc
		done   <-chan struct{}
	}
)

func Get() (*core.DefaultEngine, error) {
	once.Do(func() {
		runCtx, cancel := context.WithCancel(context.Background())
		var done <-chan struct{}
		inst, done, err = newEngine(Deps{RuntimeContext: runCtx})
		if err != nil {
			cancel()
			return
		}
		runtimeLifecycle.Lock()
		runtimeLifecycle.cancel = cancel
		runtimeLifecycle.done = done
		runtimeLifecycle.Unlock()
	})
	return inst, err
}

// Stop cancels the default engine's background profile runtime and waits for
// its worker to exit. Explicit engines created with New own their context.
func Stop(ctx context.Context) error {
	runtimeLifecycle.Lock()
	cancel := runtimeLifecycle.cancel
	done := runtimeLifecycle.done
	runtimeLifecycle.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	if done == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		runtimeLifecycle.Lock()
		if runtimeLifecycle.done == done {
			runtimeLifecycle.cancel = nil
			runtimeLifecycle.done = nil
		}
		runtimeLifecycle.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
