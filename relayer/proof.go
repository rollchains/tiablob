package relayer

import (
	"context"

	"fmt"
	"strings"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	protoblocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/rollchains/tiablob/celestia-node/share"
	"github.com/rollchains/tiablob/celestia-node/blob"
)

func (r *Relayer) GetCachedProofs() []*Proof {
	proofs := []*Proof{}

	for checkHeight := r.latestProvenHeight + 1; r.blockProofCache[checkHeight] != nil; checkHeight++ {
		proofs = append(proofs, r.blockProofCache[checkHeight])
	}

	return proofs
}

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

	squareSize := uint64(0)
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
		if len(blobs) > 0 {
			r.celestiaHeaderCache[queryHeight] = r.FetchNewHeader(ctx, queryHeight)
			block, err := r.provider.GetCelestiaBlockAtHeight(ctx, int64(queryHeight))
			if err != nil {
				fmt.Println("Error querying celestia block")
				return
			}
			squareSize = block.Block.Data.SquareSize
			fmt.Println("Square size:", squareSize)
		}
		for _, mBlob := range blobs {
			// TODO: we can aggregate adjacent blobs and prove with a single proof, fall back to individual blobs when not possible
			err = r.GetBlobProof(ctx, celestiaNodeClient, mBlob, queryHeight, squareSize)
			if err != nil {
				return
			}
		}
		r.celestiaLastQueriedHeight++
	}

	// prune old blobs and consensus states
}

// Returns nil, nil if blob in namespace is not ours
// Returns an error if we need to quit
func (r *Relayer) GetBlobProof(ctx context.Context, celestiaNodeClient *cn.Client, mBlob *blob.Blob, queryHeight uint64, squareSize uint64) error {
	// Check that the blob matches our block
	var blobBlockProto protoblocktypes.Block
	err := blobBlockProto.Unmarshal(mBlob.GetData())
	if err != nil {
		r.logger.Error(fmt.Sprintf("Error unmatshalling blob data at height: %d, %v", queryHeight, err))
		return nil
	}

	rollchainBlockHeight := blobBlockProto.Header.Height

	// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
	// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
	// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
	// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
	expectedBlock, err := r.provider.GetLocalBlockAtHeight(ctx, rollchainBlockHeight)
	if err != nil {
		r.logger.Error("Error getting block at celestia height:", queryHeight, "rollchain height", rollchainBlockHeight, "err", err)
		return nil
	}

	blockProto, err := expectedBlock.Block.ToProto()
	if err != nil {
		r.logger.Error("error expected block to proto", err.Error())
		return nil
	}
	
	blockProtoBz, err := blockProto.Marshal()
	if err != nil {
		r.logger.Error("error blockProto marshal", err.Error())
		return nil
	}

	// Replace blob data with our data for proof verification
	mBlob.Data = blockProtoBz

	shares, err := blob.BlobsToShares(mBlob)
	if err != nil {
		r.logger.Error("error BlobsToShares")
		return nil
	}

	shareIndex := GetShareIndex(uint64(mBlob.Index()), squareSize)

	// Get all shares from celestia node, confirm our shares are present
	proverShareProof, err := r.provider.ProveShares(ctx, queryHeight, shareIndex, shareIndex+uint64(len(shares)))
	if err != nil {
		r.logger.Error("error calling ProveShares", err.Error())
		return nil
	}

	proverShareProof.Data = shares // Replace with retrieved shares
	err = proverShareProof.Validate(r.celestiaHeaderCache[queryHeight].SignedHeader.Header.DataHash)
	if err != nil {
		r.logger.Error("failed verify membership, err", err.Error())
		return nil
	}

	fmt.Println("Proof added for height: ", rollchainBlockHeight)
	
	proverShareProof.Data = nil // Only include proof
	mBlob.Data = nil // Remove block
	r.blockProofCache[uint64(rollchainBlockHeight)] = &Proof{
		ShareProof: proverShareProof,
		Blob: mBlob,
		CelestiaHeight: queryHeight,
		RollchainHeight: uint64(rollchainBlockHeight),
	}

	return nil
}

func GetShareIndex(edsIndex uint64, squareSize uint64) uint64 {
	shareIndex := edsIndex
	if edsIndex > squareSize {
		shareIndex = (edsIndex - (edsIndex % squareSize)) / 2 + (edsIndex % squareSize)
	}
	return shareIndex
}