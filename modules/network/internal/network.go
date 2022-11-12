package internal

import (
	"context"
	"fmt"
	"github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/utils"
	"github.com/dan-and-dna/grpc-route"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
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
	core               *core.NetworkCore
	httpSrv            *http.Server
	httpRouter         *gin.Engine
	grpcSrv            *grpc.Server
	grpcListener       net.Listener
	listeners          map[string][]func(context.Context, interface{})                    // 协议监听者
	handlers           map[string]func(context.Context, interface{}) (interface{}, error) // 协议处理者
	ginGrpcOption      *GinGrpcOption                                                     // GinGrpc 选项
	grpcRouteOption    *GrpcRouteOption                                                   // GrpcRoute 选项
	grpcServiceDescMap map[string]*grpc.ServiceDesc                                       // grpc 服务
	isRunning          bool                                                               // 是否正在运行
	coreChanged        atomic.Bool                                                        // 配置是否更新
	mu                 sync.Mutex
}

func (network *Network) ModuleConstruct() {
	// 加载配置文件
	network.core = new(core.NetworkCore)

	network.listeners = make(map[string][]func(context.Context, interface{}))
	network.grpcServiceDescMap = make(map[string]*grpc.ServiceDesc)
	network.ginGrpcOption = new(GinGrpcOption)
	network.grpcRouteOption = new(GrpcRouteOption)
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
		network.grpcSrv = grpc.NewServer(grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			network.core.GrpcMiddlewares...,
		)))

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

func (network *Network) ListenProto(pkg, service, method string, listener func(context.Context, interface{})) {
	// TODO
}

func (network *Network) StopListenProto(pkg, service, method string) {
	// TODO
}

func (network *Network) NotifyListeners(ctx context.Context, req interface{}, key string) {
	// TODO
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

func (network *Network) NotifyHandler(ctx context.Context, req interface{}, key string) (interface{}, error) {
	if network.core.ListenHttp {
		if handler, ok := network.ginGrpcOption.GetHandler(key); ok {
			return handler.HandleProto(ctx, req)
		}
	}

	if network.core.ListenGrpc {
		if handler, ok := network.grpcRouteOption.GetHandler(key); ok {
			return handler(ctx, req)
		}
	}

	return nil, status.Error(codes.NotFound, "未知请求")
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
