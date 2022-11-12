package internal

import (
	gingrpc "github.com/dan-and-dna/gin-grpc"
	grpcroute "github.com/dan-and-dna/grpc-route"
	"github.com/gin-gonic/gin"
	"strings"
	"sync"
)

type GrpcRouteOption struct {
	Handlers map[string]grpcroute.HandleProto
	sync.RWMutex
}

func (option *GrpcRouteOption) GetHandler(key string) (grpcroute.HandleProto, bool) {
	option.RLock()
	defer option.RUnlock()

	handler, ok := option.Handlers[strings.ToLower(key)]
	if ok {
		return handler, true
	}

	return nil, false
}

func (option *GrpcRouteOption) SetHandler(key string, h grpcroute.HandleProto) {
	if key == "" || h == nil {
		return
	}

	option.RLock()
	defer option.RUnlock()

	if option.Handlers == nil {
		option.Handlers = make(map[string]grpcroute.HandleProto)
	}

	option.Handlers[key] = h
}

func (option *GrpcRouteOption) RemoveHandler(key string) {
	option.Lock()
	defer option.Unlock()

	delete(option.Handlers, key)
}

type GinGrpcOption struct {
	pathToServiceName func(*gin.Context) string
	handlers          map[string]*gingrpc.Handler
	sync.RWMutex
}

func (option *GinGrpcOption) PathToGrpcService(c *gin.Context) string {
	return option.pathToServiceName(c)
}

func (option *GinGrpcOption) GetHandler(key string) (*gingrpc.Handler, bool) {
	option.RLock()
	defer option.RUnlock()

	if handler, ok := option.handlers[key]; ok {
		return handler, true
	}
	return nil, false
}

func (option *GinGrpcOption) SetHandler(key string, handler *gingrpc.Handler) {
	option.Lock()
	defer option.Unlock()

	if option.handlers == nil {
		option.handlers = make(map[string]*gingrpc.Handler)
	}
	option.handlers[key] = handler
}

func (option *GinGrpcOption) RemoveGrpcHandler(key string) {
	option.Lock()
	defer option.Unlock()

	if option.handlers == nil {
		return
	}
	delete(option.handlers, key)
}
