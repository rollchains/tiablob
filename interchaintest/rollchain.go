package e2e

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/tiablob"
)

func GetProvenHeight(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) int64 {
	node := chain.GetNode()

	cmd := []string{"tiablob", "proven-height"}
	stdout, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)

	var height tiablob.QueryProvenHeightResponse
	cdc := chain.Config().EncodingConfig.Codec
	err = cdc.UnmarshalJSON(stdout, &height)
	require.NoError(t, err)

	return height.ProvenHeight
}

func GetPendingBlocks(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) *tiablob.QueryPendingBlocksResponse {
	node := chain.GetNode()

	cmd := []string{"tiablob", "pending-blocks"}
	stdout, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)

	var pendingBlocks tiablob.QueryPendingBlocksResponse
	cdc := chain.Config().EncodingConfig.Codec
	err = cdc.UnmarshalJSON(stdout, &pendingBlocks)
	require.NoError(t, err)

	return &pendingBlocks

}

func GetExpiredBlocks(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) *tiablob.QueryExpiredBlocksResponse {
	node := chain.GetNode()

	cmd := []string{"tiablob", "expired-blocks"}
	stdout, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)

	var expiredBlocks tiablob.QueryExpiredBlocksResponse
	cdc := chain.Config().EncodingConfig.Codec
	err = cdc.UnmarshalJSON(stdout, &expiredBlocks)
	require.NoError(t, err)

	return &expiredBlocks
}
