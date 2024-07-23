package v2

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	contypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	"github.com/rollchains/rollchain/app/upgrades"
)

// NewUpgrade constructor
func NewUpgradeV2() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          "2",
		CreateUpgradeHandler: CreateUpgradeHandlerV2,
		StoreUpgrades: storetypes.StoreUpgrades{
			Added:   []string{},
			Deleted: []string{},
		},
	}
}

func CreateUpgradeHandlerV2(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		paramsResp, err := ak.ConsensusParamsKeeper.Params(ctx, &contypes.QueryParamsRequest{})
		if err != nil {
			panic("error getting consensus params")
		}
		params := paramsResp.Params
		params.Abci.VoteExtensionsEnableHeight = plan.Height + 1
		if err = ak.ConsensusParamsKeeper.ParamsStore.Set(ctx, *params); err != nil {
			panic("error setting consensus params")
		}
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
