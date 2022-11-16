package internal

import (
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	singleInst *Grpc = nil
	once       sync.Once
)

type Grpc struct {
	core           *core.GrpcCore
	cfgMgr         *viper.Viper
	zapLogger      *zap.Logger
	coreChanged    atomic.Bool // 配置是否更新
	isModuleLoaded bool        // 模块是否加载
	isModuleRun    bool        // 模块是否运行
}

func (grpc *Grpc) ModuleConstruct() {
	// load config file

	grpc.core = new(core.GrpcCore)
	grpc.coreChanged.Store(false)

	log.Println("[grpc] constructed")
}

func (grpc *Grpc) ModuleDestruct() {
}

func (grpc *Grpc) ModuleLock() singleinstmodule.ModuleCore {
	grpc.core.Lock()
	return grpc.core
}

func (grpc *Grpc) ModuleUnlock() {
	grpc.core.Unlock()
	grpc.CoreChanged()
}

func (grpc *Grpc) ModuleUnlockTest(b *network.Bench) (*grpc.ClientConn, error) {
	grpc.core.Unlock()

	return grpc.Test(b)
}

func (grpc *Grpc) ModuleShutdown() {
	if grpc.zapLogger != nil {
		grpc.zapLogger.Sync()
	}
	log.Println("[grpc] shutdown")
}

func (grpc *Grpc) ModuleAfterRun(method string) {
	log.Printf("[grpc] %s\n", method)
	time.Sleep(50 * time.Millisecond)
}

func (grpc *Grpc) ModuleRunConfigWatcher() {
}

func (grpc *Grpc) ModuleRunStartup() {
	//grpc.CoreChanged()
}

func (grpc *Grpc) ModuleRestart() bool {
	if grpc.coreChanged.CompareAndSwap(true, false) {
		log.Println("[grpc] start restart")
		grpc.Recreate()
		return true
	}

	return false
}

func (grpc *Grpc) CoreChanged() {
	grpc.coreChanged.Store(true)
	singleinstmodule.RestartModule(grpc)
}

func (grpc *Grpc) Recreate() {
	cfg := network.ModuleLock().(*core.NetworkCore)
	defer network.ModuleUnlock()

	grpc.core.RLock()
	defer grpc.core.RUnlock()

	cfg.GrpcMiddlewares = nil

	cfg.ListenGrpc = grpc.core.Enable
	if cfg.ListenGrpc {
		cfg.ListenIp = grpc.core.ListenIp
		cfg.ListenPort = grpc.core.ListenPort
		cfg.GrpcMiddlewares = append(cfg.GrpcMiddlewares, grpc.core.Middlewares...)
	}

}

func (grpc *Grpc) Test(b *network.Bench) (*grpc.ClientConn, error) {
	cfg := network.ModuleLock().(*core.NetworkCore)

	grpc.core.RLock()
	defer grpc.core.RUnlock()

	cfg.GrpcMiddlewares = nil

	cfg.ListenGrpc = grpc.core.Enable
	cfg.GrpcMiddlewares = append(cfg.GrpcMiddlewares, grpc.core.Middlewares...)

	return network.ModuleUnlockTest(b)
}

/*
func (grpc1 *Grpc) appendTraceId() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		grpc1.traceBaseId++
		grpc_ctxtags.Extract(ctx).Set("trace_id", fmt.Sprintf("trace_%d", grpc1.traceBaseId))
		return handler(ctx, req)
	}
}
*/

func GetSingleInst() *Grpc {
	if singleInst == nil {
		once.Do(func() {
			singleInst = new(Grpc)
		})
	}

	return singleInst
}

// 注册模块
func init() {
	singleinstmodule.Register(GetSingleInst())
}
