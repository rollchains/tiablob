package e2e

import (
	"context"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

var FundsAmount = sdkmath.NewInt(1000_000000) // 1k tokens

// TestFeegrant verifies that validators can set up fee grant
func TestFeegrant(t *testing.T) {
	nv := 4
	rollchainCs := setup.RollchainChainSpec(t.Name(), nv, "local", 0)
	celestiaCs := setup.CelestiaChainSpec()
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		rollchainCs,
		celestiaCs,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	rollchainChain := chains[0].(*cosmos.CosmosChain)
	celestiaChain := chains[1].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(rollchainChain).
		AddChain(celestiaChain, ibc.WalletAmount{
			Address: "celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf",
			Amount:  sdkmath.NewInt(100_000_000_000), // 100,000 tia
			Denom:   celestiaCs.Denom,
		})

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Add feegrant wallet to broadcasting node
	_, err = celestiaChain.BuildWallet(ctx, "feegrant", "kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek")
	require.NoError(t, err)
	
	// Send a transaction to a random address to get the feegrant pub key on chain
	err = celestiaChain.SendFunds(ctx, "feegrant", ibc.WalletAmount{
		Address: "celestia1l34yq0xgya2hey0mzj3cmxcknegw2axz3dvhwt",
		Amount: sdkmath.OneInt(),
		Denom: celestiaChain.Config().Denom,
	})
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, celestiaChain))

	valCelestiaAddrs := make([]string, len(rollchainChain.Validators))
	for i, v := range rollchainChain.Validators {
		stdout, stderr, err := v.ExecBin(ctx, "keys", "tiablob", "add")
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
		t.Log("validator", i, string(stdout), string(stderr))

		stdout, stderr, err = v.ExecBin(ctx, "keys", "tiablob", "show")
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)

		valCelestiaAddrs[i] = strings.TrimSpace(string(stdout))

		if i == 0 {
			// Restore mnemonic into first val so that it can broadcast transactions
			// celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf
			stdout, stderr, err := rollchainChain.Validators[0].ExecBin(ctx, "keys", "tiablob", "restore", "-f", "kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek")
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
	stdout, stderr, err := rollchainChain.Validators[0].ExecBin(ctx, cmd...)
	require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
	t.Log(string(stdout), string(stderr))

	// faucet funds to the user
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", FundsAmount, rollchainChain)
	user := users[0]

	// balance check
	balance, err := rollchainChain.GetBalance(ctx, user.FormattedAddress(), rollchainChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, balance.Equal(FundsAmount), "user balance should be equal to fund amount")

	require.NoError(t, testutil.WaitForBlocks(ctx, 2, rollchainChain))
}
