package initialize

import (
	"context"
	"fmt"
	"sort"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Initializer 初始化器接口
type Initializer interface {
	InitializerName() string
	MigrateTable(ctx context.Context) (context.Context, error)
	TableCreated(ctx context.Context) bool
}

// orderedInitializer 有序初始化器
type orderedInitializer struct {
	order int
	init  Initializer
}

var (
	initializers []orderedInitializer
)

// RegisterInit 注册初始化器
func RegisterInit(order int, init Initializer) {
	if init == nil {
		panic("initializer cannot be nil")
	}
	initializers = append(initializers, orderedInitializer{order: order, init: init})
}

// InitializeDB 初始化数据库
func InitializeDB(db *gorm.DB) error {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	// 按顺序排序
	sort.Slice(initializers, func(i, j int) bool {
		return initializers[i].order < initializers[j].order
	})

	global.GVA_LOG.Info("开始初始化TR069-Adapter插件数据库")

	for _, init := range initializers {
		if init.init.TableCreated(ctx) {
			global.GVA_LOG.Info(fmt.Sprintf("%s 已经初始化过，跳过", init.init.InitializerName()))
			continue
		}

		global.GVA_LOG.Info(fmt.Sprintf("正在初始化 %s", init.init.InitializerName()))
		
		var err error
		ctx, err = init.init.MigrateTable(ctx)
		if err != nil {
			global.GVA_LOG.Error(fmt.Sprintf("初始化 %s 失败", init.init.InitializerName()), zap.Error(err))
			return err
		}
		
		global.GVA_LOG.Info(fmt.Sprintf("%s 初始化完成", init.init.InitializerName()))
	}

	global.GVA_LOG.Info("TR069-Adapter插件数据库初始化完成")
	return nil
}