package setup


import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// StartCelestiaAndRollchains is a helper function for quickly starting up celestia and up to 6 rollchain chains
// The max number of rollchain chain's limit is only due to the pre-defined celestia wallets
// TODO: generate celestia mnemonic/addresses as needed
func StartCelestiaAndRollchains(t *testing.T, ctx context.Context, numRollChains int) *Chains {
	rollchainChainSpecs := RollchainChainSpecs(t.Name(), numRollChains)
	celestiaChainSpec := CelestiaChainSpec()
	chainSpecs := append(rollchainChainSpecs, celestiaChainSpec)
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), chainSpecs)

	ibcChains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chains := NewChains(ibcChains)

	ic := interchaintest.NewInterchain().
		AddChain(chains.RollchainChain)

	addCelestiaWallets := []ibc.WalletAmount{
		{
			Address: celestiaWallets[0].Address,
			Amount:  math.NewInt(100_000_000_000), // 100,000 tia
			Denom:   celestiaChainSpec.Denom,
		},
	}
	for i, otherRcChain := range chains.OtherRcChains {
		ic = ic.AddChain(otherRcChain)
		addCelestiaWallets = append(addCelestiaWallets, ibc.WalletAmount{
			Address: celestiaWallets[i+1].Address,
			Amount:  math.NewInt(100_000_000_000), // 100,000 tia
			Denom:   celestiaChainSpec.Denom,

		})
	}
	ic = ic.AddChain(chains.CelestiaChain, addCelestiaWallets...)

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

	for i, rollchain := range rollchainChainSpecs {
		for j := 0; j < *rollchain.NumValidators; j++ {
			var chain *cosmos.CosmosChain
			if i == 0 {
				chain = chains.RollchainChain
			} else {
				chain = chains.OtherRcChains[i-1]
			}
			stdout, stderr, err := chain.Validators[j].ExecBin(ctx, "keys", "tiablob", "restore", celestiaWallets[i].Mnemonic)
			require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
			t.Log(string(stdout), string(stderr))
		}
	}

	fundAmount := math.NewInt(100_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chains.RollchainChain, chains.CelestiaChain)
	rollchainUser := users[0]
	celestiaUser := users[1]
	err = testutil.WaitForBlocks(ctx, 2, chains.RollchainChain, chains.CelestiaChain) // Only waiting 1 block is flaky for parachain
	require.NoError(t, err, "celestia chain failed to make blocks")

	// Check balances are correct
	rollchainUserAmount, err := chains.RollchainChain.GetBalance(ctx, rollchainUser.FormattedAddress(), chains.RollchainChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, rollchainUserAmount.Equal(fundAmount), "Initial rollchain user amount not expected")
	celestiaUserAmount, err := chains.CelestiaChain.GetBalance(ctx, celestiaUser.FormattedAddress(), chains.CelestiaChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, celestiaUserAmount.Equal(fundAmount), "Initial celestia user amount not expected")

	chains.CelestiaNode = StartCelestiaNode(t, ctx, chains.CelestiaChain, client, network)

	return chains
}