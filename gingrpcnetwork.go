package gingrpcnetwork

import (
	"github.com/dan-and-dna/singleinstmodule"

	// modules
	_ "github.com/dan-and-dna/gin-grpc-network/modules/grpc"
	_ "github.com/dan-and-dna/gin-grpc-network/modules/http"
	_ "github.com/dan-and-dna/gin-grpc-network/modules/network"
)

func Poll() {
	singleinstmodule.Run(false)
}
