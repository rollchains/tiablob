package module

import (
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	modulev1 "github.com/rollchains/tiablob/api/module/v1"
	"github.com/rollchains/tiablob/celestia-node/share"
	"github.com/rollchains/tiablob/keeper"
)

var _ appmodule.AppModule = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

func init() {
	appmodule.Register(
		new(modulev1.Module),
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Cdc          codec.Codec
	appOpts      servertypes.AppOptions
	StoreService store.KVStoreService

	StakingKeeper stakingkeeper.Keeper
	UpgradeKeeper upgradekeeper.Keeper

	storeKey storetypes.StoreKey

	publishToCelestiaBlockInterval int
	celestiaNamespace              share.Namespace
}

type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule
	Keeper *keeper.Keeper
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := keeper.NewKeeper(in.Cdc, in.appOpts, in.StoreService, &in.StakingKeeper, &in.UpgradeKeeper, in.storeKey, in.publishToCelestiaBlockInterval, in.celestiaNamespace)
	m := NewAppModule(in.Cdc, k)

	return ModuleOutputs{Module: m, Keeper: k}
}
