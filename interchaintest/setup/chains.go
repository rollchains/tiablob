package setup

import (
	"github.com/docker/docker/client"
	
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

type Chains struct {
	CelestiaChain  *cosmos.CosmosChain // celestia...
	CelestiaNode   *cosmos.SidecarProcess
	RollchainChain *cosmos.CosmosChain   // primary rollchain under test
	OtherRcChains  []*cosmos.CosmosChain // other rollchains running (could also be under test)
	Client         *client.Client
}

func NewChains(chains []ibc.Chain) *Chains {
	otherChains := make([]*cosmos.CosmosChain, len(chains)-2)
	for i := range chains {
		if i != 0 && i < len(chains)-1 {
			otherChains[i-1] = chains[i].(*cosmos.CosmosChain)
		}

	}
	return &Chains{
		CelestiaChain:  chains[len(chains)-1].(*cosmos.CosmosChain),
		RollchainChain: chains[0].(*cosmos.CosmosChain),
		OtherRcChains:  otherChains,
	}

}
