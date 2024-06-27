package blockprovider

import (
	"fmt"
	"os"

	cfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
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

// Get the vote extension enable height from genesis
// There might be a better way to get this before the first block is committed
func getGenesisVoteExtensionEnableHeight(genDoc *cmttypes.GenesisDoc, cmtConfig *cfg.Config) int64 {
	veEnableHeight := genDoc.ConsensusParams.ABCI.VoteExtensionsEnableHeight

	if veEnableHeight == 0 {
		jsonBlob, err := os.ReadFile(cmtConfig.GenesisFile())
		if err != nil {
			panic(fmt.Errorf("couldn't read GenesisDoc file: %w", err))
		}

		var genesis Genesis
		err = cmtjson.Unmarshal(jsonBlob, &genesis)
		if err == nil {
			fmt.Println("Genesis unmarshalled okay")
			veEnableHeight = genesis.Consensus.Params.ABCI.VoteExtensionsEnableHeight
		}
	}

	return veEnableHeight
}