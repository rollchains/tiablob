package e2e

import (
	"context"
	"testing"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestProveBlocksShort verifies that rollchains is proving blocks/heights (20 blocks only)
// go test -timeout 10m -v -run TestProveBlocksShort . -count 1
func TestProveBlocksShort(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
}

// TestProveBlocksLong verifies that rollchains is proving blocks/heights for 5 hours
// Test timeout should be 6 hours due to last share proof bug
// go test -timeout 6h -v -run TestProveBlocksLong . -count 1
func TestProveBlocksLong(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 9000)
}