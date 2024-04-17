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

func TestPublish(t *testing.T) {
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name()) // celestia-1-val-0-TestPublish
	//celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish

	rollchainChainSpec := DefaultChainSpec //nolint:copylockss
	nv := 2
	rollchainChainSpec.NumValidators = &nv
	rollchainChainSpec.ConfigFileOverrides = testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"rpc-url": fmt.Sprintf("http://%s:26657", celestiaAppHostname),
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
			Amount: math.NewInt(100_000_000_000), // 100,000 tia
			Denom: celestiaChainSpec.Denom,
		})

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	for i:=0; i < nv; i++ {
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

	celestiaNode := StartCelestiaNode(t, ctx, celestiaChain, client, network)
	fmt.Println("Celestia node hostname: ", celestiaNode.HostName())

	err = testutil.WaitForBlocks(ctx, 75, celestiaChain)
	require.NoError(t, err, "failed to wait for 75 blocks")

	/*valCelestiaAddrs := make([]string, len(chain.Validators))
	for i, v := range chain.Validators {
		stdout, stderr, err := v.ExecBin(ctx, "keys", "tiablob", "add")
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
		t.Log("validator", i, string(stdout), string(stderr))

		stdout, stderr, err = v.ExecBin(ctx, "keys", "tiablob", "show")
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)

		valCelestiaAddrs[i] = strings.TrimSpace(string(stdout))

		if i == 0 {
			// Restore mnemonic into first val so that it can broadcast transactions
			// celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf
			stdout, stderr, err := chain.Validators[0].ExecBin(ctx, "keys", "tiablob", "restore", "-f", "kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek")
			require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
			t.Log(string(stdout), string(stderr))
		} else {
			// Add feegranter address as offline key for all vals except the first one
			stdout, stderr, err = v.ExecBin(ctx, "keys", "tiablob", "add", "-f", "--address", "celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf")
			require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
		}

	}

	// feegrant all validators
	cmd := []string{"tx", "tiablob", "feegrant"}
	cmd = append(cmd, valCelestiaAddrs...)
	stdout, stderr, err := chain.Validators[0].ExecBin(ctx, cmd...)
	require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
	t.Log(string(stdout), string(stderr))

	// faucet funds to the user
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", GenesisFundsAmount, chain)
	user := users[0]

	// balance check
	balance, err := chain.GetBalance(ctx, user.FormattedAddress(), Denom)
	require.NoError(t, err)
	require.True(t, balance.Equal(GenesisFundsAmount), "user balance should be equal to genesis funds")

	require.NoError(t, testutil.WaitForBlocks(ctx, 100, chain))*/
}
