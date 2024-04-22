package celestia

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
This file contains the logic for storage and iteration over `IterationKey` metadata that is stored
for each consensus state. The consensus state key specified in ICS-24 and expected by counterparty chains
stores the consensus state under the key: `consensusStates/{revision_number}-{revision_height}`, with each number
represented as a string.
While this works fine for IBC proof verification, it makes efficient iteration difficult since the lexicographic order
of the consensus state keys do not match the height order of consensus states. This makes consensus state pruning and
monotonic time enforcement difficult since it is inefficient to find the earliest consensus state or to find the neighboring
consensus states given a consensus state height.
Changing the ICS-24 representation will be a major breaking change that requires counterparty chains to accept a new key format.
Thus to avoid breaking IBC, we can store a lookup from a more efficiently formatted key: `iterationKey` to the consensus state key which
stores the underlying consensus state. This efficient iteration key will be formatted like so: `iterateConsensusStates{BigEndianRevisionBytes}{BigEndianHeightBytes}`.
This ensures that the lexicographic order of iteration keys match the height order of the consensus states. Thus, we can use the SDK store's
Iterators to iterate over the consensus states in ascending/descending order by providing a mapping from `iterationKey -> consensusStateKey -> ConsensusState`.
A future version of IBC may choose to replace the ICS24 ConsensusState path with the more efficient format and make this indirection unnecessary.
*/

const KeyIterateConsensusStatePrefix = "iterateConsensusStates"

var (
	// KeyProcessedTime is appended to consensus state key to store the processed time
	KeyProcessedTime = []byte("/processedTime")
	// KeyProcessedHeight is appended to consensus state key to store the processed height
	KeyProcessedHeight = []byte("/processedHeight")
	// KeyIteration stores the key mapping to consensus state key for efficient iteration
	KeyIteration = []byte("/iterationKey")
)

// setClientState stores the client state
func setClientState(clientStore storetypes.KVStore, clientState *ClientState) {
	key := ClientStateKey()
	//val := clienttypes.MustMarshalClientState(cdc, clientState)
	val, err := json.Marshal(clientState)
	if err != nil {
		panic("Error client state marshal")
	}
	clientStore.Set(key, val)
}

// getClientState retrieves the client state from the store using the provided KVStore and codec.
// It returns the unmarshaled ClientState and a boolean indicating if the state was found.
func GetClientState(store storetypes.KVStore) (*ClientState, bool) {
	bz := store.Get(ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	var clientState ClientState
	err := json.Unmarshal(bz, &clientState)
	if err != nil {
		panic("Error client state unmarshal")
	}
	return &clientState, true
}

// setConsensusState stores the consensus state at the given height.
func setConsensusState(clientStore storetypes.KVStore, consensusState *ConsensusState, height Height) {
	key := ConsensusStateKey(height)
	val, err := json.Marshal(consensusState)
	if err != nil {
		panic("Error consensus state marshal")
	}
	clientStore.Set(key, val)
}

// GetConsensusState retrieves the consensus state from the client prefixed store.
// If the ConsensusState does not exist in state for the provided height a nil value and false boolean flag is returned
func GetConsensusState(store storetypes.KVStore, height Height) (*ConsensusState, bool) {
	bz := store.Get(ConsensusStateKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	var consensusState ConsensusState
	err := json.Unmarshal(bz, &consensusState)
	if err != nil {
		panic("Error consensus state unmarshal")
	}
	return &consensusState, true
}

// deleteConsensusState deletes the consensus state at the given height
func deleteConsensusState(clientStore storetypes.KVStore, height Height) {
	key := ConsensusStateKey(height)
	clientStore.Delete(key)
}

// ProcessedTimeKey returns the key under which the processed time will be stored in the client store.
func ProcessedTimeKey(height Height) []byte {
	return append(ConsensusStateKey(height), KeyProcessedTime...)
}

// SetProcessedTime stores the time at which a header was processed and the corresponding consensus state was created.
// This is useful when validating whether a packet has reached the time specified delay period in the tendermint client's
// verification functions
func SetProcessedTime(clientStore storetypes.KVStore, height Height, timeNs uint64) {
	key := ProcessedTimeKey(height)
	val := sdk.Uint64ToBigEndian(timeNs)
	clientStore.Set(key, val)
}

// GetProcessedTime gets the time (in nanoseconds) at which this chain received and processed a tendermint header.
// This is used to validate that a received packet has passed the time delay period.
func GetProcessedTime(clientStore storetypes.KVStore, height Height) (uint64, bool) {
	key := ProcessedTimeKey(height)
	bz := clientStore.Get(key)
	if len(bz) == 0 {
		return 0, false
	}
	return sdk.BigEndianToUint64(bz), true
}

// deleteProcessedTime deletes the processedTime for a given height
func deleteProcessedTime(clientStore storetypes.KVStore, height Height) {
	key := ProcessedTimeKey(height)
	clientStore.Delete(key)
}

// ProcessedHeightKey returns the key under which the processed height will be stored in the client store.
func ProcessedHeightKey(height Height) []byte {
	return append(ConsensusStateKey(height), KeyProcessedHeight...)
}

// SetProcessedHeight stores the height at which a header was processed and the corresponding consensus state was created.
// This is useful when validating whether a packet has reached the specified block delay period in the tendermint client's
// verification functions
func SetProcessedHeight(clientStore storetypes.KVStore, consHeight, processedHeight Height) {
	key := ProcessedHeightKey(consHeight)
	val := []byte(processedHeight.String())
	clientStore.Set(key, val)
}

// GetProcessedHeight gets the height at which this chain received and processed a tendermint header.
// This is used to validate that a received packet has passed the block delay period.
func GetProcessedHeight(clientStore storetypes.KVStore, height Height) (Height, bool) {
	key := ProcessedHeightKey(height)
	bz := clientStore.Get(key)
	if len(bz) == 0 {
		return Height{}, false
	}
	processedHeight, err := ParseHeight(string(bz))
	if err != nil {
		return Height{}, false
	}
	return processedHeight, true
}

// deleteProcessedHeight deletes the processedHeight for a given height
func deleteProcessedHeight(clientStore storetypes.KVStore, height Height) {
	key := ProcessedHeightKey(height)
	clientStore.Delete(key)
}

// IterationKey returns the key under which the consensus state key will be stored.
// The iteration key is a BigEndian representation of the consensus state key to support efficient iteration.
func IterationKey(height Height) []byte {
	heightBytes := bigEndianHeightBytes(height)
	return append([]byte(KeyIterateConsensusStatePrefix), heightBytes...)
}

// SetIterationKey stores the consensus state key under a key that is more efficient for ordered iteration
func SetIterationKey(clientStore storetypes.KVStore, height Height) {
	key := IterationKey(height)
	val := ConsensusStateKey(height)
	clientStore.Set(key, val)
}

// GetIterationKey returns the consensus state key stored under the efficient iteration key.
// NOTE: This function is currently only used for testing purposes
func GetIterationKey(clientStore storetypes.KVStore, height Height) []byte {
	key := IterationKey(height)
	return clientStore.Get(key)
}

// deleteIterationKey deletes the iteration key for a given height
func deleteIterationKey(clientStore storetypes.KVStore, height Height) {
	key := IterationKey(height)
	clientStore.Delete(key)
}

// GetHeightFromIterationKey takes an iteration key and returns the height that it references
func GetHeightFromIterationKey(iterKey []byte) Height {
	bigEndianBytes := iterKey[len([]byte(KeyIterateConsensusStatePrefix)):]
	revisionBytes := bigEndianBytes[0:8]
	heightBytes := bigEndianBytes[8:]
	revision := binary.BigEndian.Uint64(revisionBytes)
	height := binary.BigEndian.Uint64(heightBytes)
	return NewHeight(revision, height)
}

// IterateConsensusStateAscending iterates through the consensus states in ascending order. It calls the provided
// callback on each height, until stop=true is returned.
func IterateConsensusStateAscending(clientStore storetypes.KVStore, cb func(height Height) (stop bool)) {
	iterator := storetypes.KVStorePrefixIterator(clientStore, []byte(KeyIterateConsensusStatePrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		iterKey := iterator.Key()
		height := GetHeightFromIterationKey(iterKey)
		if cb(height) {
			break
		}
	}
}

// GetNextConsensusState returns the lowest consensus state that is larger than the given height.
// The Iterator returns a storetypes.Iterator which iterates from start (inclusive) to end (exclusive).
// If the starting height exists in store, we need to call iterator.Next() to get the next consenus state.
// Otherwise, the iterator is already at the next consensus state so we can call iterator.Value() immediately.
func GetNextConsensusState(clientStore storetypes.KVStore, height Height) (*ConsensusState, bool) {
	iterateStore := prefix.NewStore(clientStore, []byte(KeyIterateConsensusStatePrefix))
	iterator := iterateStore.Iterator(bigEndianHeightBytes(height), nil)
	defer iterator.Close()
	if !iterator.Valid() {
		return nil, false
	}

	// if iterator is at current height, ignore the consensus state at current height and get next height
	// if iterator value is not at current height, it is already at next height.
	if bytes.Equal(iterator.Value(), ConsensusStateKey(height)) {
		iterator.Next()
		if !iterator.Valid() {
			return nil, false
		}
	}

	csKey := iterator.Value()

	return getTmConsensusState(clientStore, csKey)
}

// GetPreviousConsensusState returns the highest consensus state that is lower than the given height.
// The Iterator returns a storetypes.Iterator which iterates from the end (exclusive) to start (inclusive).
// Thus to get previous consensus state we call iterator.Value() immediately.
func GetPreviousConsensusState(clientStore storetypes.KVStore, height Height) (*ConsensusState, bool) {
	iterateStore := prefix.NewStore(clientStore, []byte(KeyIterateConsensusStatePrefix))
	iterator := iterateStore.ReverseIterator(nil, bigEndianHeightBytes(height))
	defer iterator.Close()

	if !iterator.Valid() {
		return nil, false
	}

	csKey := iterator.Value()

	return getTmConsensusState(clientStore, csKey)
}

// PruneAllExpiredConsensusStates iterates over all consensus states for a given
// client store. If a consensus state is expired, it is deleted and its metadata
// is deleted. The number of consensus states pruned is returned.
func PruneAllExpiredConsensusStates(
	ctx sdk.Context, clientStore storetypes.KVStore,
	clientState *ClientState,
) int {
	var heights []Height

	pruneCb := func(height Height) bool {
		consState, found := GetConsensusState(clientStore, height)
		if !found { // consensus state should always be found
			return true
		}

		if clientState.IsExpired(consState.Timestamp, ctx.BlockTime()) {
			heights = append(heights, height)
		}

		return false
	}

	IterateConsensusStateAscending(clientStore, pruneCb)

	for _, height := range heights {
		deleteConsensusState(clientStore, height)
		deleteConsensusMetadata(clientStore, height)
	}

	return len(heights)
}

// Helper function for GetNextConsensusState and GetPreviousConsensusState
func getTmConsensusState(clientStore storetypes.KVStore, key []byte) (*ConsensusState, bool) {
	bz := clientStore.Get(key)
	if len(bz) == 0 {
		return nil, false
	}

	var consensusState ConsensusState
	err := json.Unmarshal(bz, &consensusState)
	if err != nil {
		return nil, false
	}

	return &consensusState, true
}

func bigEndianHeightBytes(height Height) []byte {
	heightBytes := make([]byte, 16)
	binary.BigEndian.PutUint64(heightBytes, height.GetRevisionNumber())
	binary.BigEndian.PutUint64(heightBytes[8:], height.GetRevisionHeight())
	return heightBytes
}

// setConsensusMetadata sets context time as processed time and set context height as processed height
// as this is internal tendermint light client logic.
// client state and consensus state will be set by client keeper
// set iteration key to provide ability for efficient ordered iteration of consensus states.
func setConsensusMetadata(ctx sdk.Context, clientStore storetypes.KVStore, height Height) {
	setConsensusMetadataWithValues(clientStore, height, GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano()))
}

// setConsensusMetadataWithValues sets the consensus metadata with the provided values
func setConsensusMetadataWithValues(
	clientStore storetypes.KVStore, height,
	processedHeight Height,
	processedTime uint64,
) {
	SetProcessedTime(clientStore, height, processedTime)
	SetProcessedHeight(clientStore, height, processedHeight)
	SetIterationKey(clientStore, height)
}

// deleteConsensusMetadata deletes the metadata stored for a particular consensus state.
func deleteConsensusMetadata(clientStore storetypes.KVStore, height Height) {
	deleteProcessedTime(clientStore, height)
	deleteProcessedHeight(clientStore, height)
	deleteIterationKey(clientStore, height)
}

const (
	KeyClientState          = "clientState"
	KeyConsensusStatePrefix = "consensusStates"
)

// ConsensusStatePath returns the suffix store key for the consensus state at a
// particular height stored in a client prefixed store.
func ConsensusStatePath(height Height) string {
	return fmt.Sprintf("%s/%s", KeyConsensusStatePrefix, height)
}

// ConsensusStateKey returns the store key for a the consensus state of a particular
// client stored in a client prefixed store.
func ConsensusStateKey(height Height) []byte {
	return []byte(ConsensusStatePath(height))
}

// ClientStateKey returns a store key under which a particular client state is stored
// in a client prefixed store
func ClientStateKey() []byte {
	return []byte(KeyClientState)
}

// DefaultMaxCharacterLength defines the default maximum character length used
// in validation of identifiers including the client, connection, port and
// channel identifiers.
//
// NOTE: this restriction is specific to this golang implementation of IBC. If
// your use case demands a higher limit, please open an issue and we will consider
// adjusting this restriction.
const DefaultMaxCharacterLength = 64

// ClientIdentifierValidator is the default validator function for Client identifiers.
// A valid Identifier must be between 9-64 characters and only contain alphanumeric and some allowed
// special characters (see IsValidID).
func ClientIdentifierValidator(id string) error {
	return defaultIdentifierValidator(id, 9, DefaultMaxCharacterLength)
}

// IsValidID defines regular expression to check if the string consist of
// characters in one of the following categories only:
// - Alphanumeric
// - `.`, `_`, `+`, `-`, `#`
// - `[`, `]`, `<`, `>`
var IsValidID = regexp.MustCompile(`^[a-zA-Z0-9\.\_\+\-\#\[\]\<\>]+$`).MatchString

func defaultIdentifierValidator(id string, min, max int) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid id, identifier cannot be blank")
	}
	// valid id MUST NOT contain "/" separator
	if strings.Contains(id, "/") {
		return fmt.Errorf("invalid id, identifier %s cannot contain separator '/'", id)
	}
	// valid id must fit the length requirements
	if len(id) < min || len(id) > max {
		return fmt.Errorf("invalid id, identifier %s has invalid length: %d, must be between %d-%d characters", id, len(id), min, max)
	}
	// valid id must contain only lower alphabetic characters
	if !IsValidID(id) {
		return fmt.Errorf("invalid id, identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'", id)
	}
	return nil
}