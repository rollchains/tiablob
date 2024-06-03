package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/rollchains/interchaintest/setup"
)


// go test -timeout 10m -v -run TestTiasyncFromGenesis . -count 1
func TestTiasyncFromGenesis(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	// Add a new full node, with state sync configured, it should sync from block 200
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo0",
				"sync-from-celestia": true,
			},
		},
		"config/config.toml": testutil.Toml{
			"log_level": "debug",
			"p2p": testutil.Toml{
				"laddr": "tcp://127.0.0.1:26656",
				"persistent_peers": "2b778354788120d5a0e76824ef9aa90247487480@127.0.0.1:26777",
				"persistent_peers_max_dial_period": "15s",
				"addr_book_strict": false,
				"allow_duplicate_ip": true,
				//"unconditional_peer_ids": "2b778354788120d5a0e76824ef9aa90247487480",
			},
		},
	}, 1)
	//require.NoError(t, err)
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

// go test -timeout 10m -v -run TestTiasyncResubmission . -count 1
func TestTiasyncResubmission(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	// Add a new full node, with state sync configured, it should sync from block 200
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo0",
				"sync-from-celestia": true,
			},
		},
		"config/config.toml": testutil.Toml{
			"log_level": "debug",
			"p2p": testutil.Toml{
				"laddr": "tcp://127.0.0.1:26656",
				"persistent_peers": "2b778354788120d5a0e76824ef9aa90247487480@127.0.0.1:26777",
				"persistent_peers_max_dial_period": "15s",
				"addr_book_strict": false,
				"allow_duplicate_ip": true,
				//"unconditional_peer_ids": "2b778354788120d5a0e76824ef9aa90247487480",
			},
		},
	}, 1)
	//require.NoError(t, err)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 5)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)

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
