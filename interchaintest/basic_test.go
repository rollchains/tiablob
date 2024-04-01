package e2e

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBasicChain(t *testing.T) {
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		&DefaultChainSpec,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

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

	// celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf
	stdout, stderr, err := chain.Validators[0].ExecBin(ctx, "keys", "tiablob", "restore", "kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek")
	require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
	t.Log(string(stdout), string(stderr))

	// faucet funds to the user
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", GenesisFundsAmount, chain)
	user := users[0]

	// balance check
	balance, err := chain.GetBalance(ctx, user.FormattedAddress(), Denom)
	require.NoError(t, err)
	require.True(t, balance.Equal(GenesisFundsAmount), "user balance should be equal to genesis funds")

	require.NoError(t, testutil.WaitForBlocks(ctx, 100, chain))
}
