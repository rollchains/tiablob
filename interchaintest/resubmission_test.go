package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestResubmission sets up celestia and a rollchain chains.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// go test -timeout 15m -v -run TestResubmission . -count 1
func TestResubmission(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// TestResubmission2 sets up celestia and 2 rollchain chains, each with a different namespace, both posting to Celestia. 
// Proves 20 blocks, pauses Celestia for 1 minute and resumes (repeats twice), recovering blocks that weren't posted when Celestia was down.
// go test -timeout 15m -v -run TestResubmission2 . -count 1
func TestResubmission2(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 2)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute) 
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}


// TestResubmissionHour is an hour long running test.
// It sets up celestia and 3 rollchain chains, each with a different namespace, all posting to Celestia. 
// It pauses Celestia a number of times for different lengths
// go test -timeout 60m -v -run TestResubmissionHour . -count 1
func TestResubmissionHour(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 3)

	proveXBlocks(t, ctx, chains.RollchainChain, 20)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute) 
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 10*time.Minute)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 5*time.Minute)
	pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// Expects 2 second block times
func pauseCelestiaAndRecover(t *testing.T, ctx context.Context, rollchainChain *cosmos.CosmosChain, celestiaChain *cosmos.CosmosChain, pauseTime time.Duration) {
	pauseCelestiaForX(t, ctx, celestiaChain, pauseTime)
	proveHeight := int64(pauseTime.Seconds() / 2) + 30 // Number of rollchain blocks to recover + 30 (buffer)
	proveXBlocks(t, ctx, rollchainChain, proveHeight)
}

// Prove a certain number of blocks
// Will time out after 2x + 20 blocks (2x is max catchup time + 15 blocks for polling period)
// Expects max polling period <= 30 seconds
func proveXBlocks(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, prove int64) {
	startingProvedHeight := rollchain.GetProvenHeight(t, ctx, chain)
	startingHeight, err := chain.Height(ctx)
	require.NoError(t, err)

	timeoutHeight := startingHeight + 2 * prove + 20 // Error after this height
	for provedHeight := startingProvedHeight; provedHeight < startingProvedHeight+prove; {
		provedHeight = rollchain.GetProvenHeight(t, ctx, chain)
		t.Log("Proved height: ", provedHeight)

		pendingBlocks := rollchain.GetPendingBlocks(t, ctx, chain)
		t.Log("Pending blocks: ", len(pendingBlocks.PendingBlocks))

		expiredBlocks := rollchain.GetExpiredBlocks(t, ctx, chain)
		t.Log("Expired blocks: ", len(expiredBlocks.ExpiredBlocks))
		t.Log("Current time: ", expiredBlocks.CurrentTime)

		currentHeight, err := chain.Height(ctx)
		require.NoError(t, err)

		require.True(t, currentHeight <= timeoutHeight, "proveXBlocks timed out")

		err = testutil.WaitForBlocks(ctx, 1, chain)
		require.NoError(t, err, "failed to wait for 1 block")
	}
}

func pauseCelestiaForX(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, pauseTime time.Duration) {
	err := chain.StopAllNodes(ctx)
	require.NoError(t, err)

	time.Sleep(pauseTime)

	err = chain.StartAllNodes(ctx)
	require.NoError(t, err)
}
