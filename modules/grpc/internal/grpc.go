package internal

import (
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"github.com/dan-and-dna/singleinstmodule"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func (grpc *Grpc) ModuleUnlockBenchmark(b *testing.B, method string, req, resp interface{}) {
	grpc.core.Unlock()

	grpc.Benchmark(b, method, req, resp)
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
	grpc.core.RLock()
	defer grpc.core.RUnlock()

	grpcCore := grpc.core

	cfg := network.ModuleLock().(*core.NetworkCore)
	defer network.ModuleUnlockRestart()

	cfg.GrpcMiddlewares = nil
	cfg.GrpcMiddlewaresStream = nil
	cfg.ListenGrpc = grpcCore.Enable
	if cfg.ListenGrpc {
		cfg.ListenHttp = false
		cfg.ListenIp = grpcCore.ListenIp
		cfg.ListenPort = grpcCore.ListenPort
		cfg.GrpcMiddlewares = append(cfg.GrpcMiddlewares, grpcCore.Middlewares...)
		cfg.GrpcMiddlewaresStream = append(cfg.GrpcMiddlewaresStream, grpcCore.MiddlewaresStream...)
	}
}

func (grpc *Grpc) Benchmark(b *testing.B, method string, req, resp interface{}) {
	cfg := network.ModuleLock().(*core.NetworkCore)

	grpc.core.RLock()
	cfg.ListenGrpc = true
	cfg.GrpcMiddlewares = nil
	cfg.GrpcMiddlewares = append(cfg.GrpcMiddlewares, grpc.core.Middlewares...)
	cfg.GrpcMiddlewaresStream = nil
	cfg.GrpcMiddlewaresStream = append(cfg.GrpcMiddlewaresStream, grpc.core.MiddlewaresStream...)
	grpc.core.RUnlock()

	network.ModuleUnlockBenchmark(b, method, req, resp)
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
