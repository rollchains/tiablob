package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/rollchains/tiablob"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
	storetypes2 "cosmossdk.io/store/types"
)

type Keeper struct {
	stakingKeeper *stakingkeeper.Keeper
	relayer       *tiablobrelayer.Relayer

	Validators   collections.Map[string, string]
	ClientID     collections.Item[string]
	ProvenHeight collections.Item[uint64]

	storeKey    storetypes2.StoreKey

	cdc codec.BinaryCodec
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	sk *stakingkeeper.Keeper,
	key storetypes2.StoreKey,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	return &Keeper{
		stakingKeeper: sk,

		Validators:   collections.NewMap(sb, tiablob.ValidatorsKey, "validators", collections.StringKey, collections.StringValue),
		ClientID:     collections.NewItem(sb, tiablob.ClientIDKey, "client_id", collections.StringValue),
		ProvenHeight: collections.NewItem(sb, tiablob.ProvenHeightKey, "proven_height", collections.Uint64Value),

		storeKey: key,

		cdc: cdc,
	}
}

func (k *Keeper) SetRelayer(r *tiablobrelayer.Relayer) {
	k.relayer = r
}
