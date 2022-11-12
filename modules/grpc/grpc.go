package grpc

import (
	"github.com/dan-and-dna/gin-grpc-network/modules/grpc/internal"
	"github.com/dan-and-dna/singleinstmodule"
)

type Grpc = internal.Grpc

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlock() {
	internal.GetSingleInst().ModuleUnlock()
}
