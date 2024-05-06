package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	storetypes2 "cosmossdk.io/store/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/rollchains/tiablob"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
)

type Keeper struct {
	stakingKeeper *stakingkeeper.Keeper
	upgradeKeeper *upgradekeeper.Keeper
	relayer       *tiablobrelayer.Relayer

	Validators              collections.Map[string, string]
	ClientID                collections.Item[string]
	ProvenHeight            collections.Item[uint64]
	PendingBlocksToTimeouts collections.Map[int64, int64]
	TimeoutsToPendingBlocks collections.Map[int64, PendingBlocks]

	storeKey storetypes2.StoreKey

	cdc codec.BinaryCodec

	publishToCelestiaBlockInterval int
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	sk *stakingkeeper.Keeper,
	uk *upgradekeeper.Keeper,
	key storetypes2.StoreKey,
	publishToCelestiaBlockInterval int,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	return &Keeper{
		stakingKeeper: sk,
		upgradeKeeper: uk,

		Validators:              collections.NewMap(sb, tiablob.ValidatorsKey, "validators", collections.StringKey, collections.StringValue),
		ClientID:                collections.NewItem(sb, tiablob.ClientIDKey, "client_id", collections.StringValue),
		ProvenHeight:            collections.NewItem(sb, tiablob.ProvenHeightKey, "proven_height", collections.Uint64Value),
		PendingBlocksToTimeouts: collections.NewMap(sb, tiablob.PendingBlocksToTimeouts, "pending_blocks_to_timeouts", collections.Int64Key, collections.Int64Value),
		TimeoutsToPendingBlocks: collections.NewMap(sb, tiablob.TimeoutsToPendingBlocks, "timeouts_to_pending_blocks", collections.Int64Key, codec.CollValue[PendingBlocks](cdc)),

		storeKey: key,

		cdc: cdc,

		publishToCelestiaBlockInterval: publishToCelestiaBlockInterval,
	}
}

func (k *Keeper) SetRelayer(r *tiablobrelayer.Relayer) {
	k.relayer = r
}
