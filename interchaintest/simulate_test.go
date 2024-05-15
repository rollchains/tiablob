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
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(), rollchainChainSpecs(t))

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 43000)
}

func rollchainChainSpecs(t *testing.T) []*interchaintest.ChainSpec {
	numRc := 10
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, numRc)
	for i := 0; i < numRc; i++ {
		if i == 0 {
			rollchainChainSpecs[i] = setup.RollchainChainSpec(t.Name(), 8, i, fmt.Sprintf("rc_demo%d", i))
		} else {
			rollchainChainSpecs[i] = setup.RollchainChainSpec(t.Name(), 1, i, fmt.Sprintf("rc_demo%d", i))
			if i%2 == 0 {
				rollchainChainSpecs[i].ConfigFileOverrides["config/config.toml"] = setRollchainBlockTime(3)
			}
		}
	}
	return rollchainChainSpecs
}

func celestiaChainSpec() *interchaintest.ChainSpec {
	celestiaChainSpec := setup.CelestiaChainSpec()
	celestiaChainSpec.ConfigFileOverrides = setCelestiaBlockTime(5)
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
