package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	storetypes2 "cosmossdk.io/store/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/rollchains/tiablob"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
)

type Keeper struct {
	stakingKeeper *stakingkeeper.Keeper
	relayer       *tiablobrelayer.Relayer

	Validators   collections.Map[string, string]
	ClientID     collections.Item[string]
	ProvenHeight collections.Item[uint64]
	PendingBlocksToTimeouts collections.Map[int64, int64]
	TimeoutsToPendingBlocks collections.Map[int64, []byte]

	storeKey storetypes2.StoreKey

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
		PendingBlocksToTimeouts: collections.NewMap(sb, tiablob.PendingBlocksToTimeouts, "pending_blocks_to_timeouts", collections.Int64Key, collections.Int64Value),
		TimeoutsToPendingBlocks: collections.NewMap(sb, tiablob.TimeoutsToPendingBlocks, "timeouts_to_pending_blocks", collections.Int64Key, collections.BytesValue),

		storeKey: key,

		cdc: cdc,
	}
}

func (k *Keeper) SetRelayer(r *tiablobrelayer.Relayer) {
	k.relayer = r
}
