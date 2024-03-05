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
