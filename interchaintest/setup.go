package e2e

import (
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types" // spawntag:globalfee

	globalfee "github.com/reecepbcups/globalfee/x/globalfee/types"
)

var (
	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"

	Denom   = "urc"
	Name    = "rollchain"
	ChainID = "chainid-1"
	Binary  = "rcd"

	Bech32 = "rc"

	NumberVals         = 1
	NumberFullNodes    = 0
	GenesisFundsAmount = sdkmath.NewInt(1000_000000) // 1k tokens

	ChainImage = ibc.NewDockerImage("rollchain", "local", "1025:1025")

	GasCoin = sdk.NewDecCoinFromDec(Denom, sdkmath.LegacyMustNewDecFromStr("0.0")) // spawntag:globalfee

	DefaultGenesis = []cosmos.GenesisKV{
		// default
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", VotingPeriod),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", MaxDepositPeriod),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", Denom),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		// poa: gov & testing account
		cosmos.NewGenesisKV("app_state.poa.params.admins", []string{"rc10d07y265gmmuvt4z0w9aw880jnsr700jymjvfq", "rc1hj5fveer5cjtn4wd6wstzugjfdxzl0xpc4nmns"}),
		// globalfee: set minimum fee requirements
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices", sdk.DecCoins{GasCoin}),

	}

	DefaultChainConfig = ibc.ChainConfig{
		Images: []ibc.DockerImage{
			ChainImage,
		},
		GasAdjustment: 1.5,
		ModifyGenesis: cosmos.ModifyGenesis(DefaultGenesis),
		EncodingConfig: func() *moduletestutil.TestEncodingConfig {
			cfg := cosmos.DefaultEncoding()
			// TODO: add encoding types here for the modules you want to use
			globalfee.RegisterInterfaces(cfg.InterfaceRegistry)
			return &cfg
		}(),
		Type:           "cosmos",
		Name:           Name,
		ChainID:        ChainID,
		Bin:            Binary,
		Bech32Prefix:   Bech32,
		Denom:          Denom,
		CoinType:       "118",
		GasPrices:      "0" + Denom,
		TrustingPeriod: "336h",
	}

	DefaultChainSpec = interchaintest.ChainSpec{
		Name:          Name,
		ChainName:     Name,
		Version:       ChainImage.Version,
		ChainConfig:   DefaultChainConfig,
		NumValidators: &NumberVals,
		NumFullNodes:  &NumberFullNodes,
	}
)
