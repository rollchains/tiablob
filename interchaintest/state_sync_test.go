package e2e


import (
	"context"
	"fmt"
	//"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	//"go.uber.org/zap"
	
	"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	"github.com/rollchains/rollchains/interchaintest/setup"
)

func TestStateSync(t *testing.T) {

	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*15)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 250, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt")

	provedHeight := rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)
	fmt.Println("Proved height2:", provedHeight)

	trustedHeight, trustedHash := rollchain.GetTrustedHeightAndHash(t, ctx, chains.RollchainChain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt2")


	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", "celestia-1", t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", "celestia-1", t.Name()) // celestia-1-celestia-node-0-TestPublish
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":           fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":          fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace":    "rc_demo0",
			},
		},
		"config/config.toml": testutil.Toml{
			"statesync": testutil.Toml{
				"enable": true,
				"rpc_servers": "rollchain-0-val-0-TestExport:26657,rollchain-0-val-1-TestExport:26657",
				"trust_height": trustedHeight,
				"trust_hash": trustedHash,
				"trust_period": "336h",
			},
		},
	}, 1)
	require.NoError(t, err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()
	
	err = testutil.WaitForBlocks(timeoutCtx, 55, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after halt")


}