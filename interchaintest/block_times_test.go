package e2e


import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestBlockTime4 sets up celestia and a rollchain chains.
// Rollchain has 2 sec block times (default) and Celestia has 4 sec block times
// Prove 20 blocks and the pause Celestia for 2 minutes and recover
// go test -timeout 15m -v -run TestBlockTime4 . -count 1
func TestBlockTime4(t *testing.T) {
	blockTime := 4

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = setCelestiaBlockTime(blockTime)
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	// pause and recover to submit multiple PFB in a block
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

// TestBlockTime8 sets up celestia and a rollchain chains.
// Rollchain has 2 sec block times (default) and Celestia has 8 sec block times
// Prove 20 blocks and the pause Celestia for 2 minutes and recover
// go test -timeout 15m -v -run TestBlockTime8 . -count 1
func TestBlockTime8(t *testing.T) {
	blockTime := 8

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = setCelestiaBlockTime(blockTime)
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

// TestBlockTime16 sets up celestia and a rollchain chains.
// Rollchain has 2 sec block times (default) and Celestia has 16 sec block times
// Prove 20 blocks and the pause Celestia for 2 minutes and recover
// go test -timeout 15m -v -run TestBlockTime16 . -count 1
func TestBlockTime16(t *testing.T) {
	blockTime := 16

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = setCelestiaBlockTime(blockTime)
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

func setCelestiaBlockTime(blockTime int) testutil.Toml {
	blockT := (time.Duration(blockTime) * time.Second).String()
	return testutil.Toml{
		"config/config.toml": testutil.Toml{
			"consensus": testutil.Toml{
				"timeout_commit": blockT,
				"timeout_propose": blockT,
			},
		},
	}
}