package tiablob

import (
	"cosmossdk.io/collections"
)

var (

	// ValidatorsKey saves the current validators.
	ValidatorsKey = collections.NewPrefix(0)

	// ClientIDKey saves the current clientID.
	ClientIDKey = collections.NewPrefix(1)

	// ProvenHeightKey saves the current proven height.
	ProvenHeightKey = collections.NewPrefix(2)

	// PendingBlocksToTimeouts maps pending blocks to their timeout
	PendingBlocksToTimeouts = collections.NewPrefix(3)

	// TimeoutsToPendingBlocks maps timeouts to a set of pending blocks
	TimeoutsToPendingBlocks = collections.NewPrefix(4)

	// light client store key
	ClientStoreKey = []byte("client_store/")
)

const (
	// ModuleName is the name of the module
	ModuleName = "tiablob"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querier msgs
	QuerierRoute = ModuleName

	// TransientStoreKey defines the transient store key
	TransientStoreKey = "transient_" + ModuleName

	Bech32Celestia = "celestia"

	AddrLen = 20
)
