package e2e

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	node "github.com/rollchains/rollchains/interchaintest/api/celestia-node"
	"github.com/rollchains/rollchains/interchaintest/setup"
	"github.com/rollchains/tiablob"
)

var celestiaNodeHome = "/var/cosmos-chain/celestia-node"

// TestPublish verifies on celestia node that rollchains is posting blobs, it does not check proved heights
// go test -timeout 10m -v -run TestPublish . -count 1
func TestPublish(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	celestiaNodeClient := node.NewCelestiaNodeClient(nil, chains.CelestiaNode, celestiaNodeHome)

	success := watchForPublishedBlocks(t, ctx, celestiaNodeClient, chains.RollchainChain, chains.CelestiaChain, 20)
	require.True(t, success, "failed to find all published blocks")
}

// Watch for published blocks on celestia node (no proving involved)
func watchForPublishedBlocks(
	t *testing.T,
	ctx context.Context,
	celestiaNodeClient *node.CelestiaNodeClient,
	rollchainChain *cosmos.CosmosChain,
	celestiaChain *cosmos.CosmosChain,
	publishedBlockCount int64,
) bool {
	// Run test to observe x blocks posted
	rollchainBlocksSeen := int64(0)
	rollchainHighestBlock := int64(0)
	celestiaHeight := int64(1)

	// setup time will allow publishedBlockCount to be used as a timeout
	for i := int64(0); i < publishedBlockCount; i++ {
		err := testutil.WaitForBlocks(ctx, 1, rollchainChain)
		require.NoError(t, err, "failed to wait for 1 block")

		celestiaLatestHeight, err := celestiaChain.Height(ctx)
		require.NoError(t, err, "error getting celestia height")

		for ; celestiaHeight < celestiaLatestHeight; celestiaHeight++ {
			blobs, err := celestiaNodeClient.GetAllBlobs(ctx, uint64(celestiaHeight), "0x"+hex.EncodeToString([]byte("rc_demo0")))
			require.NoError(t, err, fmt.Sprintf("error getting all blobs at height: %d, %v", celestiaHeight, err))
			t.Log("GetAllBlobs, celestia height: ", celestiaHeight)
			if len(blobs) == 0 {
				t.Log("No blobs found")
			} else {
				for j := 0; j < len(blobs); j++ {
					var signedBlock tiablob.SignedBlock
					err = signedBlock.Unmarshal(blobs[j].Data)
					if err != nil {
						t.Log("Error unmarshalling block")
					} else {
						rollchainBlocksSeen++
						if rollchainHighestBlock < signedBlock.Block.Header.Height {
							rollchainHighestBlock = signedBlock.Block.Header.Height
						}
						t.Log("Block ", signedBlock.Block.Header.Height, " found")
					}
				}
			}
		}
		// Current expectation is that blocks will not miss being published, this will change once tiablob retry logic is added
		require.Equal(t, rollchainBlocksSeen, rollchainHighestBlock, "rollchain published blocks missed")
		if rollchainHighestBlock >= publishedBlockCount {
			return true
		}
	}

	return false
}
