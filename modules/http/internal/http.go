package internal

import (
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"github.com/dan-and-dna/singleinstmodule"
	"github.com/spf13/viper"
	"log"
	"sync"
	"sync/atomic"
)

var (
	singleInst *Http = nil
	once       sync.Once
)

type Http struct {
	core           *core.HttpCore
	cfgMgr         *viper.Viper
	coreChanged    atomic.Bool // core是否更新
	isModuleLoaded bool        // 模块是否加载
	isModuleRun    bool        // 模块是否运行
}

func (http *Http) ModuleConstruct() {
	// 加载配置文件
	http.core = new(core.HttpCore)
	http.coreChanged.Store(false)

	log.Println("[http] constructed")
}

func (http *Http) ModuleAfterRun(method string) {
	log.Printf("[http] %s\n", method)
}

func (http *Http) ModuleRunConfigWatcher() {
}

func (http *Http) ModuleRunStartup() {
	//http.CoreChanged()
}

func (http *Http) ModuleDestruct() {
	http.coreChanged.Store(false)

	log.Println("[http] destructed")
}

func (http *Http) ModuleRestart() bool {
	//http.done <- struct{}{}
	if http.coreChanged.CompareAndSwap(true, false) {
		log.Println("[http] start restart")
		http.Recreate()
		return true
	}

	return false
}

func (http *Http) ModuleAfterRestart() {
	log.Println("[http] after restart")
}

func (http *Http) ModuleShutdown() {
	//http.done <- struct{}{}
	log.Println("[http] shutdown")
}

func (http *Http) ModuleLock() singleinstmodule.ModuleCore {
	http.core.Lock()
	return http.core
}

func (http *Http) ModuleUnlock() {
	http.core.Unlock()
	http.CoreChanged()
}

func (http *Http) CoreChanged() {
	// core变化 重建模块
	http.coreChanged.Store(true)
	singleinstmodule.RestartModule(http)
}

func (http *Http) Recreate() {
	cfg := network.ModuleLock().(*core.NetworkCore)
	defer network.ModuleUnlock()

	http.core.RLock()
	defer http.core.RUnlock()

	// 添加中间件
	cfg.HttpMiddlewares = nil
	cfg.HttpCtxOptions = nil

	cfg.ListenHttp = http.core.Enable
	if cfg.ListenHttp {
		cfg.ListenIp = http.core.ListenIp
		cfg.ListenPort = http.core.ListenPort
		cfg.HttpReadTimeOut = http.core.ReadTimeOut
		cfg.HttpWriteTimeOut = http.core.WriteTimeOut
		cfg.HttpPathToServiceName = http.core.PathToServiceName
		cfg.HttpPath = http.core.Path
		cfg.HttpMiddlewares = append(cfg.HttpMiddlewares, http.core.Middlewares...)
		cfg.HttpCtxOptions = append(cfg.HttpCtxOptions, http.core.CtxOptions...)
	}
}

func GetSingleInst() *Http {
	if singleInst == nil {
		once.Do(func() {
			singleInst = new(Http)
		})
	}

	return singleInst
}

// 注册到modules
func init() {
	singleinstmodule.Register(GetSingleInst())
}
