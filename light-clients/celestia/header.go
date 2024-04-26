package celestia

import (
	"bytes"
	"fmt"
	"time"

	types2 "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
)

type ClientMessage interface {
	ClientType() string
	ValidateBasic() error
}

var _ ClientMessage = (*Header)(nil)

type Header struct {
	*types2.SignedHeader `json:"signed_header,omitempty"`
	ValidatorSet         *types2.ValidatorSet `json:"validator_set,omitempty"`
	TrustedHeight        Height               `json:"trusted_height"`
	TrustedValidators    *types2.ValidatorSet `json:"trusted_validators,omitempty"`
}

// ConsensusState returns the updated consensus state associated with the header
func (h Header) ConsensusState() *ConsensusState {
	return &ConsensusState{
		Timestamp:          h.GetTime(),
		Root:               h.Header.GetDataHash(),
		NextValidatorsHash: h.Header.NextValidatorsHash,
	}
}

// ClientType defines that the Header is a Tendermint consensus algorithm
func (Header) ClientType() string {
	return ModuleName
}

// GetHeight returns the current height. It returns 0 if the tendermint
// header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetHeight() Height {
	revision := ParseChainID(h.Header.ChainID)
	return NewHeight(revision, uint64(h.Header.Height))
}

// GetTime returns the current block timestamp. It returns a zero time if
// the tendermint header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetTime() time.Time {
	return h.Header.Time
}

// ValidateBasic calls the SignedHeader ValidateBasic function and checks
// that validatorsets are not nil.
// NOTE: TrustedHeight and TrustedValidators may be empty when creating client
// with MsgCreateClient
func (h Header) ValidateBasic() error {
	if h.SignedHeader == nil {
		return fmt.Errorf("err invalid header, tendermint signed header cannot be nil")
	}
	if h.Header == nil {
		return fmt.Errorf("err invalid header, tendermint header cannot be nil")
	}
	tmSignedHeader, err := cmttypes.SignedHeaderFromProto(h.SignedHeader)
	if err != nil {
		return fmt.Errorf("err, header is not a tendermint header, %v", err)
	}
	if err := tmSignedHeader.ValidateBasic(h.Header.GetChainID()); err != nil {
		return fmt.Errorf("err, header failed basic validation, %v", err)
	}

	// TrustedHeight is less than Header for updates and misbehaviour
	if h.TrustedHeight.GTE(h.GetHeight()) {
		return fmt.Errorf("invalid header height, TrustedHeight %d must be less than header height %d", h.TrustedHeight, h.Commit.GetHeight())
	}

	if h.ValidatorSet == nil {
		return fmt.Errorf("invalid header, validator set is nil")
	}
	tmValset, err := cmttypes.ValidatorSetFromProto(h.ValidatorSet)
	if err != nil {
		return fmt.Errorf("validator set is not tendermint validator set, %v", err)
	}
	if !bytes.Equal(h.Header.ValidatorsHash, tmValset.Hash()) {
		return fmt.Errorf("invalid header, validator set does not match hash")
	}
	return nil
}
