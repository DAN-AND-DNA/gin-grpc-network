package internal

import (
	"context"
	"fmt"
	"github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/utils"
	"github.com/dan-and-dna/grpc-route"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	// 网络层单例
	singleInst *Network = nil
	once       sync.Once
)

type Network struct {
	core                  *core.NetworkCore
	httpSrv               *http.Server
	httpRouter            *gin.Engine
	grpcSrv               *grpc.Server
	grpcListener          net.Listener
	listeners             map[string][]func(context.Context, interface{})                    // 协议监听者
	handlers              map[string]func(context.Context, interface{}) (interface{}, error) // 协议处理者
	ginGrpcOption         *GinGrpcOption                                                     // GinGrpc 选项
	grpcRouteOption       *GrpcRouteOption                                                   // GrpcRoute 选项
	grpcRouteOptionStream *GrpcRouteOptionStream                                             // GrpcRouteStream 选项
	grpcServiceDescMap    map[string]*grpc.ServiceDesc                                       // grpc 服务
	isRunning             bool                                                               // 是否正在运行
	coreChanged           atomic.Bool                                                        // 配置是否更新
	mu                    sync.Mutex
}

func (network *Network) ModuleConstruct() {
	// 加载配置文件
	network.core = new(core.NetworkCore)

	network.listeners = make(map[string][]func(context.Context, interface{}))
	network.grpcServiceDescMap = make(map[string]*grpc.ServiceDesc)
	network.ginGrpcOption = new(GinGrpcOption)
	network.grpcRouteOption = new(GrpcRouteOption)
	network.grpcRouteOptionStream = new(GrpcRouteOptionStream)
	network.core.Lock()
	network.core.Enable = true
	network.core.Unlock()
	network.coreChanged.Store(false)

	log.Println("[network] constructed")
}

func (network *Network) ModuleDestruct() {
	network.coreChanged.Store(false)
	log.Println("[network] destructed")
}

func (network *Network) ModuleBeforeRun(method string) {
}

func (network *Network) ModuleShutdown() {
	network.Stop()
	log.Println("[network] shutdown")
}

func (network *Network) ModuleAfterRun(method string) {
	log.Printf("[network] %s\n", method)
}

func (network *Network) ModuleRestart() bool {
	if network.coreChanged.CompareAndSwap(true, false) {
		log.Println("[network] start restart")
		network.Stop()
		network.Recreate()
		network.Start()
		return true
	}

	return false
}

func (network *Network) ModuleAfterRestart() {
	//TODO 测试服务是否启动完毕
	time.Sleep(50 * time.Millisecond)
}

func (network *Network) ModuleRunStartup() {
	//network.CoreChanged()
}

func (network *Network) CoreChanged() {
	// 启动模块
	network.coreChanged.Store(true)
	singleinstmodule.RestartModule(network)
}

func (network *Network) ModuleLock() singleinstmodule.ModuleCore {
	network.core.Lock()
	return network.core
}

func (network *Network) ModuleUnlock() {
	network.core.Unlock()

	network.CoreChanged()
}

func (network *Network) ModuleUnlockBenchmark(b *testing.B, method string, req, resp interface{}) {
	network.core.Unlock()

	network.Benchmark(b, method, req, resp)
}

func (network *Network) Start() {
	network.core.RLock()
	defer network.core.RUnlock()

	if !network.core.Enable {
		return
	}

	if network.core.ListenGrpc {
		if network.isRunning || network.grpcListener == nil || network.grpcSrv == nil {
			return
		}

		go func() {
			if err := network.grpcSrv.Serve(network.grpcListener); err != nil {
				log.Printf("failed to serve: %v\n", err)
			}
		}()

		network.isRunning = true
		log.Printf("[network] 开始监听 %s\n", fmt.Sprintf("%s:%d", network.core.ListenIp, network.core.ListenPort))
	} else if network.core.ListenHttp {
		if network.isRunning || network.httpRouter == nil || network.httpSrv == nil {
			return
		}

		path := network.core.HttpPath
		network.httpRouter.POST(path, gingrpc.GinGrpc(network.ginGrpcOption, true, network.core.HttpCtxOptions...))

		go func() {
			if err := network.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("listen: %v\n", err)
			}
		}()

		network.isRunning = true
		log.Printf("[network] 开始监听 %s\n", fmt.Sprintf("%s:%d", network.core.ListenIp, network.core.ListenPort))
	}
}

func (network *Network) Stop() {
	// grpc 服务
	if network.grpcSrv != nil {
		network.grpcSrv.GracefulStop()
		network.grpcListener.Close()
	}

	// http 服务
	if network.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := network.httpSrv.Shutdown(ctx); err != nil {
			log.Println("[network] Server forced to shutdown: ", err)
		}
		network.httpSrv.Close()
	}
	if network.isRunning {
		log.Println("[network] 停止监听")
	}

	time.Sleep(200 * time.Millisecond)
	network.isRunning = false
}

func (network *Network) Recreate() error {
	network.core.RLock()
	defer network.core.RUnlock()

	if !network.core.Enable {
		return nil
	}
	// 清理
	network.httpSrv = nil
	network.httpRouter = nil
	network.grpcSrv = nil
	network.grpcListener = nil

	// 重建
	if network.core.ListenHttp {
		gin.SetMode(gin.ReleaseMode)
		network.httpRouter = gin.New()
		// http中间件
		network.httpRouter.Use(network.core.HttpMiddlewares...)
		network.ginGrpcOption.pathToServiceName = network.core.HttpPathToServiceName

		network.httpSrv = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", network.core.ListenIp, network.core.ListenPort),
			ReadTimeout:  time.Duration(network.core.HttpReadTimeOut) * time.Second, // 只关心 网络底层的超时，非业务侧的超时
			WriteTimeout: time.Duration(network.core.HttpWriteTimeOut) * time.Second,
			Handler:      network.httpRouter,
		}
	} else if network.core.ListenGrpc {
		var err error
		network.grpcListener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", network.core.ListenIp, network.core.ListenPort))
		if err != nil {
			return err
		}

		// grpc中间件
		network.core.GrpcMiddlewares = append(network.core.GrpcMiddlewares, grpcroute.GrpcRoute(network.grpcRouteOption), network.NoFound)
		network.core.GrpcMiddlewaresStream = append(network.core.GrpcMiddlewaresStream, grpcroute.GrpcRouteStream(network.grpcRouteOptionStream), network.NoFoundStream)
		network.grpcSrv = grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
				network.core.GrpcMiddlewares...,
			)),
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
				network.core.GrpcMiddlewaresStream...,
			)),
		)

		network.mu.Lock()
		defer network.mu.Unlock()

		for serviceName, desc := range network.grpcServiceDescMap {
			network.grpcSrv.RegisterService(desc, nil)
			log.Println("[network] 成功注册grpc服务:", serviceName)
		}
	}

	return nil
}

func (network *Network) NoFound(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return nil, status.New(codes.NotFound, "no service can help you").Err()
}

func (network *Network) NoFoundStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return status.New(codes.NotFound, "no service can help you").Err()
}

func (network *Network) ListenProto(pkg, service, method string, desc *grpc.ServiceDesc, listener func(grpc.ServerStream) error) {
	network.grpcRouteOptionStream.SetHandler(utils.MakeKey(pkg, service, method), listener)
	if desc == nil {
		return
	}

	network.mu.Lock()
	defer network.mu.Unlock()
	network.grpcServiceDescMap[desc.ServiceName] = desc
}

func (network *Network) StopListenProto(pkg, service, method string) {
	network.grpcRouteOptionStream.RemoveHandler(utils.MakeKey(pkg, service, method))
}

func (network *Network) HandleProto(pkg, service, method string, desc *grpc.ServiceDesc, handler gingrpc.Handler) {
	network.ginGrpcOption.SetHandler(utils.MakeKey(pkg, service, method), &handler)
	network.grpcRouteOption.SetHandler(utils.MakeKey(pkg, service, method), handler.HandleProto)

	if desc == nil {
		return
	}
	network.mu.Lock()
	defer network.mu.Unlock()
	network.grpcServiceDescMap[desc.ServiceName] = desc
}

func (network *Network) StopHandleProto(pkg, service, method string) {
	network.ginGrpcOption.RemoveGrpcHandler(utils.MakeKey(pkg, service, method))
	network.grpcRouteOption.RemoveHandler(utils.MakeKey(pkg, service, method))
}

type Bench struct {
	MockCh chan struct{}
	b      *testing.B
}

func NewBench(b *testing.B) *Bench {
	bc := &Bench{}
	bc.MockCh = make(chan struct{})
	bc.b = b
	return bc
}

func (bench *Bench) TagRPC(ctx context.Context, rpcTagInfo *stats.RPCTagInfo) context.Context {
	return ctx
}

func (bench *Bench) HandleRPC(ctx context.Context, s stats.RPCStats) {
	if _, ok := s.(*stats.Begin); ok {
		bench.b.StartTimer()
	} else if _, ok := s.(*stats.End); ok {
		bench.b.StopTimer()
		bench.MockCh <- struct{}{}
	}
}

func (bench *Bench) TagConn(ctx context.Context, connTagInfo *stats.ConnTagInfo) context.Context {
	return ctx
}

func (bench *Bench) HandleConn(context.Context, stats.ConnStats) {

}

func (network *Network) Benchmark(b *testing.B, method string, req, resp interface{}) {
	network.Stop()

	bc := NewBench(b)

	lis := bufconn.Listen(256 * 1024)
	network.grpcListener = lis
	network.core.GrpcMiddlewares = nil
	network.core.GrpcMiddlewares = append(network.core.GrpcMiddlewares, grpcroute.GrpcRoute(network.grpcRouteOption), network.NoFound)
	network.grpcSrv = grpc.NewServer(
		grpc.ReadBufferSize(128*1024),
		grpc.WriteBufferSize(128*1024),
		grpc.StatsHandler(stats.Handler(bc)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			network.core.GrpcMiddlewares...,
		)),
	)

	network.mu.Lock()
	for _, desc := range network.grpcServiceDescMap {
		network.grpcSrv.RegisterService(desc, nil)
	}
	network.mu.Unlock()

	go network.grpcSrv.Serve(network.grpcListener)

	var clientOpts []grpc.DialOption
	clientOpts = append(clientOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	clientOpts = append(clientOpts, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}))
	clientOpts = append(clientOpts, grpc.WithReadBufferSize(128*1024))
	clientOpts = append(clientOpts, grpc.WithWriteBufferSize(128*1024))

	conn, err := grpc.DialContext(context.Background(), "", clientOpts...)
	if err != nil {
		panic(err)
	}

	b.StopTimer()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		err := conn.Invoke(context.TODO(), method, req, resp)
		if err != nil {
			panic(err)
		}

		<-bc.MockCh
	}
}

func GetSingleInst() *Network {
	if singleInst == nil {
		once.Do(func() {
			singleInst = new(Network)
		})
	}

	return singleInst
}

func init() {
	singleinstmodule.Register(GetSingleInst())
}
