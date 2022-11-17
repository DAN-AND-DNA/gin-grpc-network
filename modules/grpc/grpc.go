package grpc

import (
	"github.com/dan-and-dna/gin-grpc-network/modules/grpc/internal"
	"github.com/dan-and-dna/singleinstmodule"
	"testing"
)

type Grpc = internal.Grpc

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlockRestart() {
	internal.GetSingleInst().ModuleUnlock()
}

func ModuleUnlockBenchmark(b *testing.B, method string, req, resp interface{}) {
	internal.GetSingleInst().ModuleUnlockBenchmark(b, method, req, resp)
}
