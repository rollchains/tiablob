package celestia

import (
	"fmt"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cometbft/cometbft/light"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var ModuleName = "celestia-da-light-client"

// NewClientState creates a new ClientState instance
func NewClientState(
	chainID string, trustLevel Fraction,
	trustingPeriod, ubdPeriod, maxClockDrift time.Duration,
	latestHeight Height,
) *ClientState {
	return &ClientState{
		ChainId:         chainID,
		TrustLevel:      trustLevel,
		TrustingPeriod:  trustingPeriod,
		UnbondingPeriod: ubdPeriod,
		MaxClockDrift:   maxClockDrift,
		LatestHeight:    latestHeight,
		FrozenHeight:    ZeroHeight(),
	}
}

// GetChainID returns the chain-id
func (cs ClientState) GetChainID() string {
	return cs.ChainId
}

// ClientType implements exported.ClientState.
func (*ClientState) ClientType() string {
	return ModuleName
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (ClientState) GetTimestampAtHeight(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return 0, fmt.Errorf("Consensus state not found, height: %s", height)
	}
	return consState.GetTimestamp(), nil
}

type Status string

const (
	Active  Status = "Active"
	Frozen  Status = "Frozen"
	Expired Status = "Expired"
	Unknown Status = "Unknown"
)

// Status returns the status of the tendermint client.
// The client may be:
// - Active: FrozenHeight is zero and client is not expired
// - Frozen: Frozen Height is not zero
// - Expired: the latest consensus state timestamp + trusting period <= current time
//
// A frozen client will become expired, so the Frozen status
// has higher precedence.
func (cs ClientState) Status(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
) Status {
	if !cs.FrozenHeight.IsZero() {
		return Frozen
	}

	// get latest consensus state from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, cs.LatestHeight)
	if !found {
		// if the client state does not have an associated consensus state for its latest height
		// then it must be expired
		return Expired
	}

	if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
		return Expired
	}

	return Active
}

// IsExpired returns whether or not the client has passed the trusting period since the last
// update (in which case no headers are considered valid).
func (cs ClientState) IsExpired(latestTimestamp, now time.Time) bool {
	expirationTime := latestTimestamp.Add(cs.TrustingPeriod)
	return !expirationTime.After(now)
}

// Validate performs a basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if strings.TrimSpace(cs.ChainId) == "" {
		return errorsmod.Wrap(ErrInvalidChainID, "chain id cannot be empty string")
	}

	// NOTE: the value of cmttypes.MaxChainIDLen may change in the future.
	// If this occurs, the code here must account for potential difference
	// between the tendermint version being run by the counterparty chain
	// and the tendermint version used by this light client.
	// https://github.com/cosmos/ibc-go/issues/177
	if len(cs.ChainId) > cmttypes.MaxChainIDLen {
		return errorsmod.Wrapf(ErrInvalidChainID, "chainID is too long; got: %d, max: %d", len(cs.ChainId), cmttypes.MaxChainIDLen)
	}

	if err := light.ValidateTrustLevel(cs.TrustLevel.ToTendermint()); err != nil {
		return err
	}
	if cs.TrustingPeriod <= 0 {
		return errorsmod.Wrap(ErrInvalidTrustingPeriod, "trusting period must be greater than zero")
	}
	if cs.UnbondingPeriod <= 0 {
		return errorsmod.Wrap(ErrInvalidUnbondingPeriod, "unbonding period must be greater than zero")
	}
	if cs.MaxClockDrift <= 0 {
		return errorsmod.Wrap(ErrInvalidMaxClockDrift, "max clock drift must be greater than zero")
	}

	// the latest height revision number must match the chain id revision number
	if cs.LatestHeight.RevisionNumber != ParseChainID(cs.ChainId) {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight,
			"latest height revision number must match chain id revision number (%d != %d)", cs.LatestHeight.RevisionNumber, ParseChainID(cs.ChainId))
	}
	if cs.LatestHeight.RevisionHeight == 0 {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight, "tendermint client's latest height revision height cannot be zero")
	}
	if cs.TrustingPeriod >= cs.UnbondingPeriod {
		return errorsmod.Wrapf(
			ErrInvalidTrustingPeriod,
			"trusting period (%s) should be < unbonding period (%s)", cs.TrustingPeriod, cs.UnbondingPeriod,
		)
	}

	return nil
}

// ZeroCustomFields returns a ClientState that is a copy of the current ClientState
// with all client customizable fields zeroed out. All chain specific fields must
// remain unchanged. This client state will be used to verify chain upgrades when a
// chain breaks a light client verification parameter such as chainID.
func (cs ClientState) ZeroCustomFields() *ClientState {
	// copy over all chain-specified fields
	// and leave custom fields empty
	return &ClientState{
		ChainId:         cs.ChainId,
		UnbondingPeriod: cs.UnbondingPeriod,
		LatestHeight:    cs.LatestHeight,
	}
}

// Initialize checks that the initial consensus state is an 07-tendermint consensus state and
// sets the client state, consensus state and associated metadata in the provided client store.
func (cs ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, consensusState *ConsensusState) error {
	SetClientState(clientStore, cdc, &cs)
	SetConsensusState(clientStore, cdc, consensusState, cs.LatestHeight)
	setConsensusMetadata(ctx, clientStore, cs.LatestHeight)

	return nil
}

// VerifyMembership is a generic proof verification method which verifies an NMT proof
// that a set of shares exist in a set of rows and a Merkle proof that those rows exist
// in a Merkle tree with a given data root.
func (cs *ClientState) VerifyMembership(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height Height, proof *ShareProof) error {
	if cs.LatestHeight.LT(height) {
		return fmt.Errorf("Invalid height, client state height < proof height (%d < %d), please ensure the client has been updated", cs.LatestHeight, height)
	}

	shareProof, err := ShareProofFromProto(proof)
	if err != nil {
		return err
	}

	consensusState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return fmt.Errorf("error consensus state not found, please ensure the proof was constructed against a height that exists on the client")
	}

	return shareProof.Validate(consensusState.GetRoot())
}
