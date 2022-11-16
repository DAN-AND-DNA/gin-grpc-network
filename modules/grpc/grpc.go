package grpc

import (
	"github.com/dan-and-dna/gin-grpc-network/modules/grpc/internal"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"github.com/dan-and-dna/singleinstmodule"
	"google.golang.org/grpc"
)

type Grpc = internal.Grpc

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlock() {
	internal.GetSingleInst().ModuleUnlock()
}

func ModuleUnlockTest(b *network.Bench) (*grpc.ClientConn, error) {
	return internal.GetSingleInst().ModuleUnlockTest(b)
}
