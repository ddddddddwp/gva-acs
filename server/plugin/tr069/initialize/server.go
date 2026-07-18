package initialize

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/handler"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func StartTR069Server() {
	if tr069Global.GlobalConfig != nil {
		global.GVA_LOG.Info("TR069 Config Check",
			zap.Bool("DumpRaw", tr069Global.GlobalConfig.DumpRaw),
			zap.Int("DumpMaxBytes", tr069Global.GlobalConfig.DumpMaxBytes),
			zap.Bool("InfoLogEnable", tr069Global.GlobalConfig.InfoLogEnable),
			zap.String("InfoLogDir", tr069Global.GlobalConfig.InfoLogDir),
		)
		// Force print to stdout to ensure visibility even if logger is file-only
		fmt.Printf("\n[TR069-DEBUG] Config Loaded - DumpRaw: %v InfoLogEnable: %v InfoLogDir: %s\n",
			tr069Global.GlobalConfig.DumpRaw,
			tr069Global.GlobalConfig.InfoLogEnable,
			tr069Global.GlobalConfig.InfoLogDir,
		)
	} else {
		global.GVA_LOG.Error("TR069 GlobalConfig is nil")
		fmt.Println("\n[TR069-DEBUG] GlobalConfig is nil")
	}

	addr := tr069Global.GlobalConfig.Address
	if addr == "" {
		addr = ":7458" // Default port
	}

	engine := SetupEngine()

	go func() {
		global.GVA_LOG.Info("Starting TR069 Server", zap.String("address", addr))
		if err := engine.Run(addr); err != nil {
			global.GVA_LOG.Error("TR069 Server failed to start", zap.Error(err))
		}
	}()
}

// SetupEngine creates and configures the gin engine for TR069
// Exported for testing purposes
func SetupEngine() *gin.Engine {
	routes, workers, objectStore, err := buildRuntimeFileIngressRoutes()
	if err != nil {
		panic(fmt.Errorf("initialize TR-069 file ingress: %w", err))
	}
	setTransferWorkers(workers)
	setArtifactStore(objectStore)
	return setupEngine(routes)
}

func setupEngine(fileIngressRoutes map[string]gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	// TR069 调试辅助：为每个请求生成仅用于内部日志链路的 traceId。
	// 删除/禁用：移除这一行即可，不影响核心 TR069 处理逻辑。
	engine.Use(middleware.EnsureTraceID())
	engine.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}
		if param.Latency > time.Minute {
			param.Latency = param.Latency.Truncate(time.Second)
		}
		cwmpID := ""
		if param.Keys != nil {
			if v, ok := param.Keys["cwmpId"]; ok {
				if s, ok := v.(string); ok {
					cwmpID = s
				}
			}
		}
		if len(cwmpID) > 12 {
			cwmpID = cwmpID[:12]
		}
		reqPart := ""
		if cwmpID != "" {
			reqPart = " | cwmp:" + cwmpID
		}
		return fmt.Sprintf(
			"[TR069] %s |%s %3d %s| %13v | %15s |%s %-7s %s %s%s\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			statusColor,
			param.StatusCode,
			resetColor,
			param.Latency,
			param.ClientIP,
			methodColor,
			param.Method,
			param.Path,
			resetColor,
			reqPart,
			param.ErrorMessage,
		)
	}))

	// Raw XML capture is deliberately limited to CWMP routes. File ingress bodies
	// must never be buffered or written to the XML/raw diagnostic log.
	cwmp := engine.Group("")
	cwmp.Use(middleware.RawDump(middleware.RawDumpConfig{}))
	cwmp.Use(middleware.RawResponseDump(middleware.RawResponseDumpConfig{}))
	cwmp.POST("/", handler.CWMPHandler)
	cwmp.POST("/acs", handler.CWMPHandler)

	for routePath, routeHandler := range fileIngressRoutes {
		engine.Any(routePath, routeHandler)
	}

	return engine
}

func buildRuntimeFileIngressRoutes() (map[string]gin.HandlerFunc, *service.TransferWorkers, service.ArtifactStore, error) {
	runtime := config.CurrentRuntime()
	if !runtime.Settings.FileIngress.Enabled {
		return nil, nil, nil, nil
	}
	if global.GVA_DB == nil {
		return nil, nil, nil, errors.New("database is required")
	}
	if global.GVA_REDIS == nil {
		return nil, nil, nil, errors.New("Redis is required")
	}
	storeConfig := runtime.Settings.FileIngress.ArtifactStore
	objectStore, err := adapter.NewMinioArtifactStoreClient(
		storeConfig.Endpoint, storeConfig.AccessKey, storeConfig.SecretKey,
		storeConfig.Bucket, storeConfig.UseSSL,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	transferStore := service.NewTransferStore(global.GVA_DB)
	identityStore := adapter.NewUploadIdentityStore(global.GVA_REDIS)
	deviceResolver := service.NewUploadDeviceResolver(global.GVA_DB, transferStore, identityStore, runtime.FileIngress.IdentityBindingTTL)
	receiver := service.NewTransferReceiver(transferStore, objectStore)
	workers := service.NewTransferWorkers(transferStore, objectStore)
	authenticator := middleware.NewFileAuthenticator(middleware.RuntimeFileCredentialProvider{}, adapter.NewRedisDigestNonceStore(global.GVA_REDIS))
	ingressHandler := handler.NewFileIngressHandler(authenticator, deviceResolver, receiver, handler.RuntimeFileIngressChannelProvider{})
	routes := make(map[string]gin.HandlerFunc)
	for _, channel := range runtime.Settings.FileIngress.Channels {
		if channel.Enabled {
			routes[channel.Path] = ingressHandler
		}
	}
	return routes, workers, objectStore, nil
}

var transferWorkerRuntime struct {
	sync.Mutex
	workers *service.TransferWorkers
}

var artifactStoreRuntime struct {
	sync.RWMutex
	store service.ArtifactStore
}

func setTransferWorkers(workers *service.TransferWorkers) {
	transferWorkerRuntime.Lock()
	transferWorkerRuntime.workers = workers
	transferWorkerRuntime.Unlock()
}

func setArtifactStore(store service.ArtifactStore) {
	artifactStoreRuntime.Lock()
	artifactStoreRuntime.store = store
	artifactStoreRuntime.Unlock()
}

// CurrentArtifactStore returns the object store initialized for file ingress.
// It may be nil when file ingress is disabled; metadata list APIs remain usable.
func CurrentArtifactStore() service.ArtifactStore {
	artifactStoreRuntime.RLock()
	defer artifactStoreRuntime.RUnlock()
	return artifactStoreRuntime.store
}

func StartTransferWorkers(ctx context.Context) {
	transferWorkerRuntime.Lock()
	workers := transferWorkerRuntime.workers
	transferWorkerRuntime.Unlock()
	if workers != nil {
		go workers.Run(ctx)
	}
}

func StopTransferWorkers(ctx context.Context) error {
	transferWorkerRuntime.Lock()
	workers := transferWorkerRuntime.workers
	transferWorkerRuntime.Unlock()
	if workers == nil {
		return nil
	}
	return workers.Stop(ctx)
}
