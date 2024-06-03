package setup

import (
	"fmt"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types" // spawntag:globalfee

	globalfee "github.com/strangelove-ventures/globalfee/x/globalfee/types"
)

var (
	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"

	Denom  = "urc"
	Binary = "rcd"
	Bech32 = "rc"

	GenesisFundsAmount = sdkmath.NewInt(1000_000000) // 1k tokens

	GasCoin = sdk.NewDecCoinFromDec(Denom, sdkmath.LegacyMustNewDecFromStr("0.0")) // spawntag:globalfee
)

// RollchainChainSpecs is a helper function for setting up 1 or more rollchains, use the RollchainChainSpec helper function for more control.
func RollchainChainSpecs(testName string, numRc int) []*interchaintest.ChainSpec {
	chainSpecs := make([]*interchaintest.ChainSpec, numRc)
	for i := 0; i < numRc; i++ {
		if i == 0 {
			chainSpecs[i] = RollchainChainSpec(testName, 2, i, fmt.Sprintf("rc_demo%d", i), 0)
		} else {
			chainSpecs[i] = RollchainChainSpec(testName, 1, i, fmt.Sprintf("rc_demo%d", i), 0)
		}
	}
	return chainSpecs
}

// Set up default rollchain chain spec with custom values
// testName: to generate celestia's app and node hostnames
// numVals: each chain will want this custom (non-primaries expected to be 1)
func RollchainChainSpec(testName string, numVals int, index int, namespace string, pubInterval int) *interchaintest.ChainSpec {
	NumberFullNodes := 0
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, testName)            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, testName) // celestia-1-celestia-node-0-TestPublish

	chainID := fmt.Sprintf("rollchain-%d", index)
	name := fmt.Sprintf("Rollchain%d", index)
	chainImage := ibc.NewDockerImage("rollchain", "local", "1025:1025")

	defaultGenesis := []cosmos.GenesisKV{
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

	defaultChainConfig := ibc.ChainConfig{
		Images: []ibc.DockerImage{
			chainImage,
		},
		GasAdjustment: 1.5,
		ModifyGenesis: cosmos.ModifyGenesis(defaultGenesis),
		EncodingConfig: func() *moduletestutil.TestEncodingConfig {
			cfg := cosmos.DefaultEncoding()
			// TODO: add encoding types here for the modules you want to use
			globalfee.RegisterInterfaces(cfg.InterfaceRegistry)
			return &cfg
		}(),
		Type:           "cosmos",
		Name:           name,
		ChainID:        chainID,
		Bin:            Binary,
		Bech32Prefix:   Bech32,
		Denom:          Denom,
		CoinType:       "118",
		GasPrices:      "0" + Denom,
		TrustingPeriod: "336h",
		ConfigFileOverrides: testutil.Toml{
			"config/app.toml": testutil.Toml{
				"celestia": testutil.Toml{
					"app-rpc-url":           fmt.Sprintf("http://%s:26657", celestiaAppHostname),
					"node-rpc-url":          fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
					"override-namespace":    namespace,
					"override-pub-interval": pubInterval,
				},
				"state-sync": testutil.Toml{
					"snapshot-interval":    100,
					"snapshot-keep-recent": 2,
				},
			},
		},
		/*HostPortOverride: map[int]int{
			26657: 26657,
			1317: 1317,
			9090: 9090,
			26656: 26656,
			1234: 1234,
			26777: 26777,
		},*/
	}

	return &interchaintest.ChainSpec{
		Name:          name,
		ChainName:     name,
		Version:       chainImage.Version,
		ChainConfig:   defaultChainConfig,
		NumValidators: &numVals,
		NumFullNodes:  &NumberFullNodes,
	}
}
