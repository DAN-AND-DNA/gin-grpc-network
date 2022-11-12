package network

import (
	"context"
	gingrpc "github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/gin-grpc-network/modules/network/internal"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc"
)

type Network = internal.Network

func NotifyListeners(ctx context.Context, req interface{}, key string) {
	internal.GetSingleInst().NotifyListeners(ctx, req, key)
}

func ListenProto(pkg, service, method string, listener func(context.Context, interface{})) {
	internal.GetSingleInst().ListenProto(pkg, service, method, listener)
}

func StopListenProto(pkg, service, method string) {
	internal.GetSingleInst().StopListenProto(pkg, service, method)
}

func HandleProto(pkg, service, method string, desc *grpc.ServiceDesc, handler gingrpc.Handler) {
	internal.GetSingleInst().HandleProto(pkg, service, method, desc, handler)
}

func StopHandleProto(pkg, service, method string) {
	internal.GetSingleInst().StopHandleProto(pkg, service, method)
}

func NotifyHandler(ctx context.Context, req interface{}, key string) (interface{}, error) {
	return internal.GetSingleInst().NotifyHandler(ctx, req, key)
}

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlock() {
	internal.GetSingleInst().ModuleUnlock()
}
