package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestSimulateRealNetwork behaves as close as possible to a real network, only scaled a bit faster (update this test as needed)
// 10 rollchain chains run with 2 or 3 second block times, the first has 8 validators, others have 1 validator (increases celestia txs)
// Celestia runs with 5 second blocks time (actual is 12 seconds with other chains usually sub-6 second block times)
// Each rollchain will have its own namespace (no namespace collisions), 5 block publishing interval
// No Celestia downtime for this test
// go test -timeout 25h -v -run TestSimulateRealNetwork . -count 1
func TestSimulateRealNetwork(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(5), rollchainChainSpecs(t))

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 43000)
}

// TestSimulateRealBlockTime uses 12 second blocks for celestia and 5 second blocks for rollchains
// Each rollchain will have its own namespace (no namespace collisions), 5 block publishing interval
// No Celestia downtime for this test
// go test -timeout 15m -v -run TestSimulateRealBlockTimes . -count 1
func TestSimulateRealBlockTimes(t *testing.T) {
	ctx := context.Background()

	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 2)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo0", 0, "0")
	rollchainChainSpecs[0].ConfigFileOverrides["config/config.toml"] = setRollchainBlockTime(5)
	rollchainChainSpecs[1] = setup.RollchainChainSpec(t.Name(), 2, 1, "rc_demo1", 0, "0")
	rollchainChainSpecs[1].ConfigFileOverrides["config/config.toml"] = setRollchainBlockTime(5)
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(12), rollchainChainSpecs)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 100)
}

func rollchainChainSpecs(t *testing.T) []*interchaintest.ChainSpec {
	numRc := 10
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, numRc)
	for i := 0; i < numRc; i++ {
		if i == 0 {
			rollchainChainSpecs[i] = setup.RollchainChainSpec(t.Name(), 8, i, fmt.Sprintf("rc_demo%d", i), 0, "0")
		} else {
			rollchainChainSpecs[i] = setup.RollchainChainSpec(t.Name(), 1, i, fmt.Sprintf("rc_demo%d", i), 0, "0")
			if i%2 == 0 {
				rollchainChainSpecs[i].ConfigFileOverrides["config/config.toml"] = setRollchainBlockTime(3)
			}
		}
	}
	return rollchainChainSpecs
}

func celestiaChainSpec(blockTime int) *interchaintest.ChainSpec {
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = setCelestiaBlockTime(blockTime)
	return celestiaChainSpec
}

func setRollchainBlockTime(blockTime int) testutil.Toml {
	blockT := (time.Duration(blockTime) * time.Second).String()
	return testutil.Toml{
		"consensus": testutil.Toml{
			"timeout_commit":  blockT,
			"timeout_propose": blockT,
		},
	}
}
