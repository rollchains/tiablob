package e2e


import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestResubmission sets up celestia and a rollchain chains.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// go test -timeout 15m -v -run TestResubmission$ . -count 1
func TestBlockTime4(t *testing.T) {
	blockTime := 4
	blockT := (time.Duration(blockTime) * time.Second).String()

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = testutil.Toml{
		"config/config.toml": testutil.Toml{
			"consensus": testutil.Toml{
				"timeout_commit": blockT,
				"timeout_propose": blockT,
			},
		},
	}
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	// pause and recover to submit multiple PFB in a block
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

func TestBlockTime8(t *testing.T) {
	blockTime := 8
	blockT := (time.Duration(blockTime) * time.Second).String()

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = testutil.Toml{
		"config/config.toml": testutil.Toml{
			"consensus": testutil.Toml{
				"timeout_commit": blockT,
				"timeout_propose": blockT,
			},
		},
	}
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

func TestBlockTime16(t *testing.T) {
	blockTime := 16
	blockT := (time.Duration(blockTime) * time.Second).String()

	ctx := context.Background()
	rollchainChainSpecs := setup.RollchainChainSpecs(t.Name(), 1)
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = testutil.Toml{
		"config/config.toml": testutil.Toml{
			"consensus": testutil.Toml{
				"timeout_commit": blockT,
				"timeout_propose": blockT,
			},
		},
	}
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, rollchainChainSpecs)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}