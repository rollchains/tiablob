package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	
	"github.com/rollchains/tiablob"
)

// Semi-interesting, not all too accurate metrics
type Metrics struct {
	MaxProvenHeight int64
	PreviousProvenHeightChange time.Time
	MaxTimeToProveHeight time.Duration
	MinTimeToProveHeight time.Duration
	MaxPendingBlocks int
	MaxExpiredBlocks int
}

func NewMetrics() *Metrics {
	return &Metrics{
		MinTimeToProveHeight: time.Hour,
	}
}

// Expects 2 second block times
func (m *Metrics) pauseCelestiaAndRecover(t *testing.T, ctx context.Context, rollchainChain *cosmos.CosmosChain, celestiaChain *cosmos.CosmosChain, pauseTime time.Duration) {
	m.pauseCelestiaForX(t, ctx, celestiaChain, pauseTime)
	proveHeight := int64(pauseTime.Seconds() / 2) + 30 // Number of rollchain blocks to recover + 30 (buffer)
	m.proveXBlocks(t, ctx, rollchainChain, proveHeight)
}

// Prove a certain number of blocks
// Timeout is the 2x (# of blocks to prove) + 45 blocks (publishing timeout) + 15 blocks (max polling period)
// Timeout is generous, but needed due to this bug: https://github.com/celestiaorg/celestia-app/pull/2208
// Expects max polling period <= 30 seconds
func (m *Metrics) proveXBlocks(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, prove int64) {
	startingProvedHeight := rollchain.GetProvenHeight(t, ctx, chain)
	startingHeight, err := chain.Height(ctx)
	require.NoError(t, err)

	timeoutHeight := startingHeight + 2 * prove + 60 // Error after this height
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

		m.injestState(provedHeight, pendingBlocks, expiredBlocks)

		require.True(t, currentHeight <= timeoutHeight, "proveXBlocks timed out")

		err = testutil.WaitForBlocks(ctx, 1, chain)
		require.NoError(t, err, "failed to wait for 1 block")
	}
}

func (m *Metrics) pauseCelestiaForX(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, pauseTime time.Duration) {
	err := chain.StopAllNodes(ctx)
	require.NoError(t, err)

	time.Sleep(pauseTime)

	err = chain.StartAllNodes(ctx)
	require.NoError(t, err)
}

func (m *Metrics) injestState(provedHeight int64, pendingBlocks *tiablob.QueryPendingBlocksResponse, expiredBlocks *tiablob.QueryExpiredBlocksResponse) {
	provenHeightChanged := false
	previousProvenHeight := int64(0)
	if m.MaxProvenHeight < provedHeight {
		previousProvenHeight = m.MaxProvenHeight
		provenHeightChanged = true
		m.MaxProvenHeight = provedHeight
	}

	numPendingBlocks := len(pendingBlocks.PendingBlocks)
	if m.MaxPendingBlocks < numPendingBlocks {
		m.MaxPendingBlocks = numPendingBlocks
	}

	numExpiredBlocks := len(expiredBlocks.ExpiredBlocks)
	if m.MaxExpiredBlocks < numExpiredBlocks {
		m.MaxExpiredBlocks = numExpiredBlocks
	}

	if provenHeightChanged {
		currentTime := time.Now()
		if previousProvenHeight != 0 {
			timeDiff := currentTime.Sub(m.PreviousProvenHeightChange)
			if m.MaxTimeToProveHeight < timeDiff {
				m.MaxTimeToProveHeight = timeDiff
			}
			if m.MinTimeToProveHeight > timeDiff {
				m.MinTimeToProveHeight = timeDiff
			}
		}
		m.PreviousProvenHeightChange = currentTime
	}
}

func (m *Metrics) PrintMetrics(t *testing.T) {
	t.Log("Metrics: ", 
		"MaxProvenHeight=", m.MaxProvenHeight,
		"MaxPendingBlocks=", m.MaxPendingBlocks,
		"MaxExpiredBlocks=", m.MaxExpiredBlocks,
		"MaxTimeToProveHeight", m.MaxTimeToProveHeight,
		"MinTimeToProveHeight", m.MinTimeToProveHeight,
	)
}