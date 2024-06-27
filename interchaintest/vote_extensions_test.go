package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	//"cosmossdk.io/math"
	//"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	//"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	"github.com/rollchains/rollchains/interchaintest/setup"
)

// go test -timeout 10m -v -run TestTiasyncVeFromGenesis . -count 1
func TestTiasyncVeFromGenesis(t *testing.T) {
	ctx := context.Background()
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 1)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0, "1")
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(2), rollchainChainSpecs)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	// Add a new full node that syncs from celestia
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
				"laddr":          "tcp://0.0.0.0:26656",
				"upstream-peers": chains.RollchainChain.Nodes().PeerString(ctx),
			},
		},
		"config/config.toml": testutil.Toml{
			//"log_level": "debug",
			"p2p": testutil.Toml{
				"laddr":              "tcp://127.0.0.1:26777",
				"persistent_peers":   "",
				"addr_book_strict":   false,
				"allow_duplicate_ip": true,
				"pex":                false,
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(timeoutCtx, 100, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}
}

// TestStateSync produces >210 blocks (2 snapshots), gets trusted height/hash for fullnode, starts/syncs fullnode with statesync
// verifies all nodes are producing blocks and the state sync'd fullnode does not have blocks earlier than the snapshot height
// go test -timeout 20m -v -run TestTiasyncStateSync . -count 1
/*func TestTiasyncStateSync(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*15)
	defer timeoutCtxCancel()

	// Wait 210 blocks for 2 snapshots, 100 block interval
	err := testutil.WaitForBlocks(timeoutCtx, 210, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt")

	// Get a trusted height and hash, wait for 10 blocks for chain to proceed
	trustedHeight, trustedHash := rollchain.GetTrustedHeightAndHash(t, ctx, chains.RollchainChain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt2")

	// Add a new full node, with state sync configured, it should sync from block 200
	// Add a new full node, with state sync configured, it should sync from block 200
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	rollchainVal0Hostname := fmt.Sprintf("%s-val-0-%s", chains.RollchainChain.Config().ChainID, t.Name())
	rollchainVal1Hostname := fmt.Sprintf("%s-val-1-%s", chains.RollchainChain.Config().ChainID, t.Name())
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo0",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
				"laddr":          "tcp://0.0.0.0:26656",
				"upstream-peers": chains.RollchainChain.Nodes().PeerString(ctx),
			},
		},
		"config/config.toml": testutil.Toml{
			//"log_level": "debug",
			"statesync": testutil.Toml{
				"enable":       true,
				"rpc_servers":  fmt.Sprintf("%s:26657,%s:26657", rollchainVal0Hostname, rollchainVal1Hostname),
				"trust_height": trustedHeight,
				"trust_hash":   trustedHash,
				"trust_period": "1h",
			},
			"p2p": testutil.Toml{
				"laddr":              "tcp://127.0.0.1:26777",
				"persistent_peers":   "",
				"addr_book_strict":   false,
				"allow_duplicate_ip": true,
				"pex":                false,
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	// wait for 30 blocks, new full node should be fully caught up
	err = testutil.WaitForBlocks(timeoutCtx, 100, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt")

	// Ensure all nodes have a height greater than 250 (sum of WaitForBlocks, 210+10+30)
	// Full node will state sync from block 200, there should be 5-10 pending blocks, if it gets a few blocks passed 200, we're good
	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}

	// Test assumes no fullnodes sync'd from genesis
	// verify that a block before the snapshot height (200) does not exist
	height := int64(150)
	for _, fn := range chains.RollchainChain.FullNodes {
		_, err = fn.Client.Block(ctx, &height)
		require.Error(t, err)
		require.Contains(t, err.Error(), "height 150 is not available, lowest height is 201")
	}
}*/