package relayer

import (
	types2 "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/light-clients/celestia"
	celestiatypes "github.com/tendermint/tendermint/types"
)

type InjectClientData struct {
	CreateClient *CreateClient `json:"create_client,omitempty"`
	Headers      []*Header     `json:"headers,omitempty"`
	Proofs       []*Proof      `json:"proofs,omitempty"`
}

func (d InjectClientData) IsEmpty() bool {
	if d.CreateClient != nil || len(d.Headers) != 0 || len(d.Proofs) != 0 {
		return false
	}
	return true
}

type CreateClient struct {
	ClientState    celestia.ClientState
	ConsensusState celestia.ConsensusState
}

type Header struct {
	*types2.SignedHeader `json:"signed_header,omitempty"`
	ValidatorSet         []byte          `json:"validator_set,omitempty"` // can't use json
	TrustedHeight        celestia.Height `json:"trusted_height"`
	TrustedValidators    []byte          `json:"trusted_validators,omitempty"` //can't use json
}

type Proof struct {
	ShareProof      *celestiatypes.ShareProof
	Blob            *blob.Blob
	CelestiaHeight  uint64 // Consensus state height
	RollchainHeight uint64
}
