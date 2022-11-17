package network

import (
	gingrpc "github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/gin-grpc-network/modules/network/internal"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc"
	"testing"
)

type Network = internal.Network
type Bench = internal.Bench

func ListenProto(pkg, service, method string, desc *grpc.ServiceDesc, listener func(grpc.ServerStream) error) {
	internal.GetSingleInst().ListenProto(pkg, service, method, desc, listener)
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

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlockRestart() {
	internal.GetSingleInst().ModuleUnlock()
}

func ModuleUnlockBenchmark(b *testing.B, method string, req, resp interface{}) {
	internal.GetSingleInst().ModuleUnlockBenchmark(b, method, req, resp)
}
