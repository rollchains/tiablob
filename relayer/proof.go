package relayer

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	blocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/rollchains/tiablob/celestia-node/share"
)

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context) {
	celestiaNodeClient, err := cn.NewClient(ctx, r.nodeRpcUrl, r.nodeAuthToken)
	if err != nil {
		r.logger.Error("Error creating celestia node client", "error", err)
		return
	}
	defer celestiaNodeClient.Close()
	
	celestiaLatestHeight, err := r.provider.QueryLatestHeight(ctx)
	if err != nil {
	 	r.logger.Error("Error querying latest height from Celestia", "error", err)
		return
	}

	txClient := txtypes.NewServiceClient(r.clientCtx)
	for r.celestiaLastQueriedHeight < celestiaLatestHeight {
		// get the namespace blobs from that height
		queryHeight := uint64(r.celestiaLastQueriedHeight+1)
		blobs, err := celestiaNodeClient.Blob.GetAll(ctx, queryHeight, []share.Namespace{r.celestiaNamespace.Bytes()})
		if err != nil {
			if strings.Contains(err.Error(), "blob: not found") {
				r.celestiaLastQueriedHeight++
				continue
			}
			r.logger.Error(fmt.Sprintf("Error on GetAll at height: %d, %v", queryHeight, err))
			return
		}
		for _, blob := range blobs {
			// Check that the blob matches our block
			var blobBlock blocktypes.Block
			err = blobBlock.Unmarshal(blob.GetData())
			if err != nil {
				r.logger.Error(fmt.Sprintf("Error unmatshalling blob data at height: %d, %v", queryHeight, err))
				continue
			}
			expectedBlock, err := txClient.GetBlockWithTxs(ctx, &txtypes.GetBlockWithTxsRequest{Height: blobBlock.Header.Height})
			if err != nil {
				r.logger.Error("Error getting block at celestia height:", queryHeight, "chain height", blobBlock.Header.Height, "err", err)
				continue
			}
			// Just verify hashes for now, but expand this, maybe checksum it?
			if !reflect.DeepEqual(blobBlock.Header.AppHash, expectedBlock.Block.Header.AppHash) {
				r.logger.Error("App hash mismatch")
				continue
			}
			// Get proof
			proof, err := celestiaNodeClient.Blob.GetProof(ctx, queryHeight, r.celestiaNamespace.Bytes(), blob.Commitment)
			if err != nil {
				r.logger.Error(fmt.Sprintf("Error on GetProof at height: %d, %v", queryHeight, err))
				return
			}
			included, err := celestiaNodeClient.Blob.Included(ctx, queryHeight, r.celestiaNamespace.Bytes(), proof, blob.Commitment)
			if err != nil {
				r.logger.Error("Err on included")
				return
			}
			if !included {
				r.logger.Error("Not included")
				return
			}
			r.blockProofCache[uint64(blobBlock.Header.Height)] = proof
			fmt.Println("Proof added for height: ", blobBlock.Header.Height)
		}
		r.celestiaLastQueriedHeight++
	}
}
