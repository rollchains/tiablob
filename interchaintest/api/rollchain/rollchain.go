package rollchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/tiablob"
)

// Rollchain/tiablob APIs

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

func GetTrustedHeightAndHash(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) (int, string) {
	node := chain.GetNode()

	cmd := []string{"status", "--node", fmt.Sprintf("tcp://%s:26657", node.HostName())}
	stdout, _, err := node.ExecBin(ctx, cmd...)
	require.NoError(t, err)

	fmt.Println("GetState:", string(stdout))

	var status Status
	err = json.Unmarshal(stdout, &status)
	require.NoError(t, err)

	fmt.Println("Height:", status.SyncInfo.LatestBlockHeight)
	fmt.Println("Hash:", status.SyncInfo.LatestBlockHash)

	height, err := strconv.Atoi(status.SyncInfo.LatestBlockHeight)
	require.NoError(t, err)

	return height, status.SyncInfo.LatestBlockHash
}

type Status struct {
	SyncInfo SyncInfo `json:"sync_info"`
}

type SyncInfo struct {
	LatestBlockHash string `json:"latest_block_hash"`
	LatestBlockHeight string `json:"latest_block_height"`
}