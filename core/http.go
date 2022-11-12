package core

import (
	gingrpc "github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/singleinstmodule"
	"github.com/gin-gonic/gin"
)

type HttpCore struct {
	singleinstmodule.SingleInstModuleCore

	Enable            bool                      // 是否启动模块
	ListenIp          string                    // http 监听ip
	ListenPort        int                       // http 监听端口
	ReadTimeOut       int                       // 读超时
	WriteTimeOut      int                       // 写超时
	PathToServiceName func(*gin.Context) string // path 转成 grpc 服务
	Path              string                    // path
	Middlewares       []gin.HandlerFunc         // http中间件
	CtxOptions        []gingrpc.GrpcCtxOption
}
