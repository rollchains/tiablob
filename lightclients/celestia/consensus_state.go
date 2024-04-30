package celestia

import (
	"fmt"
	"time"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"
)

// SentinelRoot is used as a stand-in root value for the consensus state set at the upgrade height
const SentinelRoot = "sentinel_root"

// NewConsensusState creates a new ConsensusState instance.
func NewConsensusState(
	timestamp time.Time, root []byte, nextValsHash cmtbytes.HexBytes,
) *ConsensusState {
	return &ConsensusState{
		Timestamp:          timestamp,
		Root:               root,
		NextValidatorsHash: nextValsHash,
	}
}

func (ConsensusState) ClientType() string {
	return ModuleName
}

// GetRoot returns the commitment Root for the specific
func (cs ConsensusState) GetRoot() []byte {
	return cs.Root
}

// GetTimestamp returns block time in nanoseconds of the header that created consensus state
func (cs ConsensusState) GetTimestamp() uint64 {
	return uint64(cs.Timestamp.UnixNano())
}

// ValidateBasic defines a basic validation for the tendermint consensus state.
// NOTE: ProcessedTimestamp may be zero if this is an initial consensus state passed in by relayer
// as opposed to a consensus state constructed by the chain.
func (cs ConsensusState) ValidateBasic() error {
	if len(cs.Root) == 0 {
		return fmt.Errorf("err invalid consensus, root cannot be empty")
	}
	if err := cmttypes.ValidateHash(cs.NextValidatorsHash); err != nil {
		return fmt.Errorf("error, next validator hash is invalid")
	}
	if cs.Timestamp.Unix() <= 0 {
		return fmt.Errorf("err invalid consensus, timestamp must be a positive Unix time")
	}
	return nil
}
