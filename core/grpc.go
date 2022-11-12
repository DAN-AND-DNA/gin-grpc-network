package core

import (
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc"
)

type GrpcCore struct {
	singleinstmodule.SingleInstModuleCore

	Enable      bool   // 是否启动模块
	ListenIp    string // http 监听ip
	ListenPort  int    // http 监听端口
	Middlewares []grpc.UnaryServerInterceptor
}
