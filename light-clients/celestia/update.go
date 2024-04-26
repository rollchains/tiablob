package celestia

import (
	"bytes"
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/light"
	cmttypes "github.com/cometbft/cometbft/types"
)

// VerifyClientMessage checks if the clientMessage is of type Header or Misbehaviour and verifies the message
func (cs *ClientState) VerifyClientMessage(
	ctx sdk.Context, clientStore storetypes.KVStore,
	clientMsg ClientMessage,
) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, msg)
	case *Misbehaviour:
		return cs.verifyMisbehaviour(ctx, clientStore, msg)
	default:
		return fmt.Errorf("invalid client type")
	}
}

// verifyHeader returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
func (cs *ClientState) verifyHeader(
	ctx sdk.Context, clientStore storetypes.KVStore,
	header *Header,
) error {
	currentTimestamp := ctx.BlockTime()

	// Retrieve trusted consensus states for each Header in misbehaviour
	consState, found := GetConsensusState(clientStore, header.TrustedHeight)
	if !found {
		return fmt.Errorf("consensus state not found, could not get trusted consensus state from clientStore for Header at TrustedHeight: %s", header.TrustedHeight)
	}

	if err := checkTrustedHeader(header, consState); err != nil {
		return err
	}

	// UpdateClient only accepts updates with a header at the same revision
	// as the trusted consensus state
	if header.GetHeight().GetRevisionNumber() != header.TrustedHeight.RevisionNumber {
		return errorsmod.Wrapf(
			ErrInvalidHeaderHeight,
			"header height revision %d does not match trusted header revision %d",
			header.GetHeight().GetRevisionNumber(), header.TrustedHeight.RevisionNumber,
		)
	}

	tmTrustedValidators, err := cmttypes.ValidatorSetFromProto(header.TrustedValidators)
	if err != nil {
		return errorsmod.Wrap(err, "trusted validator set in not tendermint validator set type")
	}

	tmSignedHeader, err := cmttypes.SignedHeaderFromProto(header.SignedHeader)
	if err != nil {
		return errorsmod.Wrap(err, "signed header in not tendermint signed header type")
	}

	tmValidatorSet, err := cmttypes.ValidatorSetFromProto(header.ValidatorSet)
	if err != nil {
		return errorsmod.Wrap(err, "validator set in not tendermint validator set type")
	}

	// assert header height is newer than consensus state
	if header.GetHeight().LTE(header.TrustedHeight) {
		return fmt.Errorf("invalid header, header height ≤ consensus state height (%s ≤ %s)", header.GetHeight(), header.TrustedHeight)
	}

	// Construct a trusted header using the fields in consensus state
	// Only Height, Time, and NextValidatorsHash are necessary for verification
	// NOTE: updates must be within the same revision
	trustedHeader := cmttypes.Header{
		ChainID:            cs.GetChainID(),
		Height:             int64(header.TrustedHeight.RevisionHeight),
		Time:               consState.Timestamp,
		NextValidatorsHash: consState.NextValidatorsHash,
	}
	signedHeader := cmttypes.SignedHeader{
		Header: &trustedHeader,
	}

	// Verify next header with the passed-in trustedVals
	// - asserts trusting period not passed
	// - assert header timestamp is not past the trusting period
	// - assert header timestamp is past latest stored consensus state timestamp
	// - assert that a TrustLevel proportion of TrustedValidators signed new Commit
	err = light.Verify(
		&signedHeader,
		tmTrustedValidators, tmSignedHeader, tmValidatorSet,
		cs.TrustingPeriod, currentTimestamp, cs.MaxClockDrift, cs.TrustLevel.ToTendermint(),
	)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	return nil
}

// UpdateState expects a Tendermint header as clientMsg and performs 07-tendermint UpdateState
// logic, updating the client state to the new header and returning a list containing the updated
// consensus height. Please note that the commitmment root of the stored consensus state is the data
// hash of the header (as opposed to the header app hash, in vanilla 07-tendermint logic).

// UpdateState may be used to either create a consensus state for:
// - a future height greater than the latest client state height
// - a past height that was skipped during bisection
// If we are updating to a past height, a consensus state is created for that height to be persisted in client store
// If we are updating to a future height, the consensus state is created and the client state is updated to reflect
// the new latest height
// A list containing the updated consensus height is returned.
// UpdateState must only be used to update within a single revision, thus header revision number and trusted height's revision
// number must be the same. To update to a new revision, use a separate upgrade path
// UpdateState will prune the oldest consensus state if it is expired.
func (cs ClientState) UpdateState(ctx sdk.Context, clientStore storetypes.KVStore, clientMsg ClientMessage) []Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &Header{}, clientMsg))
	}

	cs.pruneOldestConsensusState(ctx, clientStore)

	// check for duplicate update
	if _, found := GetConsensusState(clientStore, header.GetHeight()); found {
		// perform no-op
		return []Height{header.GetHeight()}
	}

	height := header.GetHeight()
	if height.GT(cs.LatestHeight) {
		cs.LatestHeight = height
	}

	consensusState := &ConsensusState{
		Timestamp:          header.GetTime(),
		Root:               header.Header.GetDataHash(),
		NextValidatorsHash: header.Header.NextValidatorsHash,
	}

	// set client state, consensus state and associated metadata
	setClientState(clientStore, &cs)
	setConsensusState(clientStore, consensusState, header.GetHeight())
	setConsensusMetadata(ctx, clientStore, header.GetHeight())

	return []Height{height}
}

// pruneOldestConsensusState will retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs ClientState) pruneOldestConsensusState(ctx sdk.Context, clientStore storetypes.KVStore) {
	// Check the earliest consensus state to see if it is expired, if so then set the prune height
	// so that we can delete consensus state and all associated metadata.
	var (
		pruneHeight Height
	)

	pruneCb := func(height Height) bool {
		consState, found := GetConsensusState(clientStore, height)
		// this error should never occur
		if !found {
			panic(fmt.Errorf("consensus state not found, failed to retrieve consensus state at height: %s", height))
		}

		if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
			pruneHeight = height
		}

		return true
	}

	IterateConsensusStateAscending(clientStore, pruneCb)

	// if pruneHeight is set, delete consensus state and metadata
	if pruneHeight.RevisionHeight != 0 && pruneHeight.RevisionNumber != 0 {
		deleteConsensusState(clientStore, pruneHeight)
		deleteConsensusMetadata(clientStore, pruneHeight)
	}
}

// UpdateStateOnMisbehaviour updates state upon misbehaviour, freezing the ClientState. This method should only be called when misbehaviour is detected
// as it does not perform any misbehaviour checks.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, clientStore storetypes.KVStore, _ ClientMessage) {
	cs.FrozenHeight = FrozenHeight

	bz, err := json.Marshal(&cs)
	if err != nil {
		panic("error on client state marshal in update state on misbehaviour")
	}

	clientStore.Set(ClientStateKey(), bz)
}

// checkTrustedHeader checks that consensus state matches trusted fields of Header
func checkTrustedHeader(header *Header, consState *ConsensusState) error {
	tmTrustedValidators, err := cmttypes.ValidatorSetFromProto(header.TrustedValidators)
	if err != nil {
		return errorsmod.Wrap(err, "trusted validator set in not tendermint validator set type")
	}

	// assert that trustedVals is NextValidators of last trusted header
	// to do this, we check that trustedVals.Hash() == consState.NextValidatorsHash
	tvalHash := tmTrustedValidators.Hash()
	if !bytes.Equal(consState.NextValidatorsHash, tvalHash) {
		return errorsmod.Wrapf(
			ErrInvalidValidatorSet,
			"trusted validators %s, does not hash to latest trusted validators. Expected: %X, got: %X",
			header.TrustedValidators, consState.NextValidatorsHash, tvalHash,
		)
	}
	return nil
}
