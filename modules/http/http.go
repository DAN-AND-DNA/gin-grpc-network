package http

import (
	"github.com/dan-and-dna/gin-grpc-network/modules/http/internal"
	"github.com/dan-and-dna/singleinstmodule"
)

func ModuleLock() singleinstmodule.ModuleCore {
	return internal.GetSingleInst().ModuleLock()
}

func ModuleUnlock() {
	internal.GetSingleInst().ModuleUnlock()
}
