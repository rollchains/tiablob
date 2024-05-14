package e2e

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestProveBlocks verifies that rollchains is proving blocks/heights
func TestProveBlocks(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	for provedHeight := int64(0); provedHeight < 20; {
		provedHeight = rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)
		t.Log("Proved height: ", provedHeight)

		pendingBlocks := rollchain.GetPendingBlocks(t, ctx, chains.RollchainChain)
		t.Log("Pending blocks: ", len(pendingBlocks.PendingBlocks))

		expiredBlocks := rollchain.GetExpiredBlocks(t, ctx, chains.RollchainChain)
		t.Log("Expired blocks: ", len(expiredBlocks.ExpiredBlocks))
		t.Log("Current time: ", expiredBlocks.CurrentTime)

		err := testutil.WaitForBlocks(ctx, 1, chains.RollchainChain)
		require.NoError(t, err, "failed to wait for 1 block")
	}
}
