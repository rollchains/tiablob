package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestNamespaceCollision starts up 2 rollchains with the same namespace posting to celestia
// Other than attempting to prove blobs from the other chain, it should behave smoothly, only additional overhead
// go test -timeout 12m -v -run TestNamespaceCollision . -count 1
func TestNamespaceCollision(t *testing.T) {
	rollchainChainSpecs := []*interchaintest.ChainSpec{
		setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo"),
		setup.RollchainChainSpec(t.Name(), 2, 1, "rc_demo"),
	}

	ctx := context.Background()
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(), rollchainChainSpecs)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	proveXBlocks(t, ctx, chains.RollchainChain, 20)
}