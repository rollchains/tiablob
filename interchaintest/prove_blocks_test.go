package e2e

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestProveBlocks(t *testing.T) {
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish

	rollchainChainSpec := DefaultChainSpec //nolint:copylockss
	nv := 2
	rollchainChainSpec.NumValidators = &nv
	rollchainChainSpec.ConfigFileOverrides = testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":  fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url": fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
			},
		},
	}
	celestiaChainSpec := CelestiaChainSpec //nolint:copylockss
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		&rollchainChainSpec,
		&celestiaChainSpec,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	rollchainChain := chains[0].(*cosmos.CosmosChain)
	celestiaChain := chains[1].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(rollchainChain).
		AddChain(celestiaChain, ibc.WalletAmount{
			Address: "celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf",
			Amount:  math.NewInt(100_000_000_000), // 100,000 tia
			Denom:   celestiaChainSpec.Denom,
		})

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
		//BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	for i := 0; i < nv; i++ {
		// celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf
		stdout, stderr, err := rollchainChain.Validators[i].ExecBin(ctx, "keys", "tiablob", "restore", "kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek")
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
		t.Log(string(stdout), string(stderr))
	}

	fundAmount := math.NewInt(100_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, rollchainChain, celestiaChain)
	rollchainUser := users[0]
	celestiaUser := users[1]
	err = testutil.WaitForBlocks(ctx, 2, rollchainChain, celestiaChain) // Only waiting 1 block is flaky for parachain
	require.NoError(t, err, "celestia chain failed to make blocks")

	// Check balances are correct
	rollchainUserAmount, err := rollchainChain.GetBalance(ctx, rollchainUser.FormattedAddress(), rollchainChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, rollchainUserAmount.Equal(fundAmount), "Initial rollchain user amount not expected")
	celestiaUserAmount, err := celestiaChain.GetBalance(ctx, celestiaUser.FormattedAddress(), celestiaChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, celestiaUserAmount.Equal(fundAmount), "Initial celestia user amount not expected")

	_ = StartCelestiaNode(t, ctx, celestiaChain, client, network)

	for provedHeight := int64(0) ; provedHeight < 20 ; {
		provedHeight = GetProvenHeight(t, ctx, rollchainChain)
		t.Log("Proved height: ", provedHeight)

		pendingBlocks := GetPendingBlocks(t, ctx, rollchainChain)
		t.Log("Pending blocks: ", len(pendingBlocks.PendingBlocks))

		expiredBlocks := GetExpiredBlocks(t, ctx, rollchainChain)
		t.Log("Expired blocks: ", len(expiredBlocks.ExpiredBlocks))
		t.Log("Current time: ", expiredBlocks.CurrentTime)

		err := testutil.WaitForBlocks(ctx, 1, rollchainChain)
		require.NoError(t, err, "failed to wait for 1 block")

	}
}
