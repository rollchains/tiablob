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


// go test -timeout 10m -v -run TestTiasync . -count 1
func TestTiasync(t *testing.T) {
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
				"persistent_peers": "2b778354788120d5a0e76824ef9aa90247487480@127.0.0.1:26777",
				"seeds": "2b778354788120d5a0e76824ef9aa90247487480@127.0.0.1:26777",
				//"persistent_peers": "2b778354788120d5a0e76824ef9aa90247487480@rollchain-0-fn-0-TestTiasync:26777",
				//"seeds": "2b778354788120d5a0e76824ef9aa90247487480@rollchain-0-fn-0-TestTiasync:26777",
				"addr_book_strict": false,
				"allow_duplicate_ip": true,
				"unconditional_peer_ids": "2b778354788120d5a0e76824ef9aa90247487480",
			},
		},
	}, 1)
	require.NoError(t, err)
	//require.Error(t, err) // expected to fail, no peers
	//fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	/*
	// wait for 10 blocks, new full node should be fully caught up
	err = testutil.WaitForBlocks(timeoutCtx, 1, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	fnID, err := chains.RollchainChain.FullNodes[0].NodeID(ctx)
	require.NoError(t, err)

	fnHostname := chains.RollchainChain.FullNodes[0].HostName()

	err = chains.RollchainChain.FullNodes[0].StopContainer(ctx)
	require.NoError(t, err)
	err = chains.RollchainChain.FullNodes[0].RemoveContainer(ctx)
	require.NoError(t, err)

	fn := chains.RollchainChain.FullNodes[0]
	configFileOverrides := testutil.Toml{
		"config/config.toml": testutil.Toml{
			"p2p": testutil.Toml{
				"persistent_peers": fmt.Sprintf("%s@%s:26777", fnID, fnHostname),
				"seeds": fmt.Sprintf("%s@%s:26777", fnID, fnHostname),
				"allow_duplicate_ip": true,
				"unconditional_peer_ids": fnID,
			},
		},
	}
	for configFile, modifiedConfig := range configFileOverrides {
		modifiedToml, ok := modifiedConfig.(testutil.Toml)
		require.True(t, ok)
		err = testutil.ModifyTomlConfigFile(
			ctx,
			zaptest.NewLogger(t),
			fn.DockerClient,
			fn.TestName,
			fn.VolumeName,
			configFile,
			modifiedToml,
		)
		require.NoError(t, err)
	}

	err = chains.RollchainChain.FullNodes[0].CreateNodeContainer(ctx)
	require.NoError(t, err)
	err = chains.RollchainChain.FullNodes[0].StartContainer(ctx)
	require.NoError(t, err)
	*/

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
