package blockprovider

import (
	"fmt"
	"os"

	cfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
	sm "github.com/cometbft/cometbft/state"
)

type Genesis struct {
	Consensus Consensus `json:"consensus"`
}

type Consensus struct {
	Params Params `json:"params"`
}

type Params struct {
	ABCI cmttypes.ABCIParams `json:"abci"`
}

// Get the vote extension enable height from state and if 0, try from genesis
func getInitialVoteExtensionEnableHeight(genDoc *cmttypes.GenesisDoc, cmtConfig *cfg.Config, state sm.State) int64 {
	veEnableHeight := state.ConsensusParams.ABCI.VoteExtensionsEnableHeight
	
	if veEnableHeight == 0 {
		veEnableHeight = genDoc.ConsensusParams.ABCI.VoteExtensionsEnableHeight
	}

	if veEnableHeight == 0 {
		jsonBlob, err := os.ReadFile(cmtConfig.GenesisFile())
		if err != nil {
			panic(fmt.Errorf("couldn't read GenesisDoc file: %w", err))
		}

		var genesis Genesis
		err = cmtjson.Unmarshal(jsonBlob, &genesis)
		if err == nil {
			veEnableHeight = genesis.Consensus.Params.ABCI.VoteExtensionsEnableHeight
		}
	}

	return veEnableHeight
}