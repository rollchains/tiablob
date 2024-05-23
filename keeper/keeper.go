package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	storetypes2 "cosmossdk.io/store/types"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/celestia-node/share"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
)

type Keeper struct {
	stakingKeeper *stakingkeeper.Keeper
	upgradeKeeper *upgradekeeper.Keeper
	relayer       *tiablobrelayer.Relayer

	Validators              collections.Map[string, string]
	ClientID                collections.Item[string]
	ProvenHeight            collections.Item[int64]
	PendingBlocksToTimeouts collections.Map[int64, int64]
	TimeoutsToPendingBlocks collections.Map[int64, PendingBlocks]

	storeKey storetypes2.StoreKey

	cdc codec.BinaryCodec

	publishToCelestiaBlockInterval int
	injectedProofsLimit            int
	celestiaNamespace              share.Namespace

	proposerAddress []byte
}

func NewKeeper(
	cdc codec.BinaryCodec,
	appOpts servertypes.AppOptions,
	storeService storetypes.KVStoreService,
	sk *stakingkeeper.Keeper,
	uk *upgradekeeper.Keeper,
	key storetypes2.StoreKey,
	publishToCelestiaBlockInterval int,
	celestiaNamespace share.Namespace,
) *Keeper {
	cfg := tiablobrelayer.CelestiaConfigFromAppOpts(appOpts)
	sb := collections.NewSchemaBuilder(storeService)

	if cfg.OverrideNamespace != "" {
		var err error
		celestiaNamespace, err = share.NewBlobNamespaceV0([]byte(cfg.OverrideNamespace))
		if err != nil {
			panic(err)
		}
	}

	return &Keeper{
		stakingKeeper: sk,
		upgradeKeeper: uk,

		Validators:              collections.NewMap(sb, tiablob.ValidatorsKey, "validators", collections.StringKey, collections.StringValue),
		ClientID:                collections.NewItem(sb, tiablob.ClientIDKey, "client_id", collections.StringValue),
		ProvenHeight:            collections.NewItem(sb, tiablob.ProvenHeightKey, "proven_height", collections.Int64Value),
		PendingBlocksToTimeouts: collections.NewMap(sb, tiablob.PendingBlocksToTimeouts, "pending_blocks_to_timeouts", collections.Int64Key, collections.Int64Value),
		TimeoutsToPendingBlocks: collections.NewMap(sb, tiablob.TimeoutsToPendingBlocks, "timeouts_to_pending_blocks", collections.Int64Key, codec.CollValue[PendingBlocks](cdc)),

		storeKey: key,

		cdc: cdc,

		publishToCelestiaBlockInterval: publishToCelestiaBlockInterval,
		injectedProofsLimit:            cfg.MaxFlushSize,
		celestiaNamespace:              celestiaNamespace,
	}
}

func (k *Keeper) SetRelayer(r *tiablobrelayer.Relayer) {
	k.relayer = r
}
