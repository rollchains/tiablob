package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/rollchains/tiablob"
)

var V2Upgrade = Upgrade{
		UpgradeName:          "2",
		CreateUpgradeHandler: V2UpgradeHandler,
		StoreUpgrades: storetypes.StoreUpgrades{
			Added:   []string{
				tiablob.StoreKey,
			},
			Deleted: []string{},
		},
	}

func V2UpgradeHandler(
	mm ModuleManager,
	configurator module.Configurator,
	ak *AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ak.TiablobKeeper.SetProvenHeight(ctx, plan.Height)
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
