package relayer

import (
	"context"
	"strings"

	protoblocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/celestia-node/share"
	"github.com/rollchains/tiablob/lightclients/celestia"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
)

func (r *Relayer) getCachedProof(height int64) *celestia.BlobProof {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.blockProofCache[height]
}

func (r *Relayer) setCachedProof(height int64, proof *celestia.BlobProof) {
	r.mu.Lock()
	r.blockProofCache[height] = proof
	r.mu.Unlock()
}

// GetCachedProofs() returns the next set of proofs to verify on-chain
func (r *Relayer) GetCachedProofs(proofLimit int, latestProvenHeight int64) []*celestia.BlobProof {
	var proofs []*celestia.BlobProof
	checkHeight := latestProvenHeight + 1
	proof := r.getCachedProof(checkHeight)
	for proof != nil {
		proofs = append(proofs, proof)
		if len(proofs) >= proofLimit {
			break
		}
		checkHeight++
		proof = r.getCachedProof(checkHeight)
	}

	return proofs
}

func (r *Relayer) HasCachedProof(block int64) bool {
	if proof := r.getCachedProof(block); proof != nil {
		return true
	}
	return false
}

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal
// that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context, latestClientState *celestia.ClientState) {
	if latestClientState == nil {
		return
	}

	celestiaNodeClient, err := cn.NewClient(ctx, r.nodeRpcUrl, r.nodeAuthToken)
	if err != nil {
		r.logger.Error("creating celestia node client", "error", err)
		return
	}
	defer celestiaNodeClient.Close()

	celestiaLatestHeight, err := r.celestiaProvider.QueryLatestHeight(ctx)
	if err != nil {
		r.logger.Error("querying latest height from Celestia", "error", err)
		return
	}

	squareSize := uint64(0)

	for queryHeight := r.celestiaLastQueriedHeight + 1; queryHeight < celestiaLatestHeight; queryHeight++ {
		// get the namespace blobs from that height
		blobs, err := celestiaNodeClient.Blob.GetAll(ctx, uint64(queryHeight), []share.Namespace{r.celestiaNamespace.Bytes()})
		if err != nil {
			// this error just indicates we don't have a blob at this height
			if strings.Contains(err.Error(), "blob: not found") {
				r.celestiaLastQueriedHeight = queryHeight
				continue
			}
			r.logger.Error("Celestia node blob getall", "height", queryHeight, "error", err)
			return
		}
		if len(blobs) > 0 {
			block, err := r.celestiaProvider.GetBlockAtHeight(ctx, queryHeight)
			if err != nil {
				r.logger.Error("querying celestia block", "height", queryHeight, "error", err)
				return
			}
			squareSize = block.Block.Data.SquareSize
		}
		for _, mBlob := range blobs {
			// TODO: we can aggregate adjacent blobs and prove with a single proof, fall back to individual blobs when not possible
			err = r.GetBlobProof(ctx, celestiaNodeClient, mBlob, queryHeight, squareSize, latestClientState)
			if err != nil {
				r.logger.Error("getting blob proof", "error", err)
				return
			}
		}
		r.celestiaLastQueriedHeight = queryHeight
	}
}

// GetBlobProof
//   - unmarshals blob into block
//   - fetches same block from this node, converts to proto type, and marshals
//   - fetches proof from celestia app
//   - verifies proof using block data from this node
//   - stores proof in cache
//
// returns an error only when we shouldn't hit an error, otherwise nil if the blob could be someone elses (i.e. same namespace used)
func (r *Relayer) GetBlobProof(ctx context.Context, celestiaNodeClient *cn.Client, mBlob *blob.Blob, queryHeight int64, squareSize uint64, latestClientState *celestia.ClientState) error {
	var blobBlockProto protoblocktypes.Block
	err := blobBlockProto.Unmarshal(mBlob.GetData())
	if err != nil {
		r.logger.Error("blob unmarshal", "height", queryHeight, "error", err)
		return nil
	}

	rollchainBlockHeight := blobBlockProto.Header.Height

	// Ignore any blocks that are <= the latest proven height
	if rollchainBlockHeight <= r.latestProvenHeight {
		return nil
	}

	// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
	// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
	// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
	// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
	expectedBlock, err := r.localProvider.GetBlockAtHeight(ctx, rollchainBlockHeight)
	if err != nil {
		r.logger.Error("getting local block", "celestia height", queryHeight, "rollchain height", rollchainBlockHeight, "err", err)
		return nil
	}

	blockProto, err := expectedBlock.Block.ToProto()
	if err != nil {
		return err
	}

	blockProtoBz, err := blockProto.Marshal()
	if err != nil {
		return err
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
	proverShareProof, err := r.celestiaProvider.ProveShares(ctx, uint64(queryHeight), shareIndex, shareIndex+uint64(len(shares)))
	if err != nil {
		r.logger.Error("error calling ProveShares", "error", err)
		return nil
	}

	// Get the header to verify the proof
	header := r.getCachedHeader(queryHeight)
	if header == nil {
		header = r.FetchNewHeader(ctx, queryHeight, latestClientState)
	}

	proverShareProof.Data = shares // Replace with retrieved shares
	err = proverShareProof.Validate(header.SignedHeader.Header.DataHash)
	if err != nil {
		r.logger.Error("failed verify membership", "error", err)
		return nil
	}

	// We have a valid proof, if we haven't cached the header, cache it
	// This goroutine is the only way to delete headers, so it should be kept that way
	if r.getCachedHeader(queryHeight) == nil {
		r.setCachedHeader(queryHeight, header)
	}

	proverShareProof.Data = nil // Only include proof
	mBlob.Data = nil            // Only include blob metadata
	shareProofProto := celestia.TmShareProofToProto(proverShareProof)

	r.setCachedProof(rollchainBlockHeight, &celestia.BlobProof{
		ShareProof:      shareProofProto,
		Blob:            celestia.BlobToProto(mBlob),
		CelestiaHeight:  queryHeight,
		RollchainHeight: rollchainBlockHeight,
	})

	return nil
}

// GetShareIndex calculates the share index given the EDS index of the blob and square size of the respective block
func GetShareIndex(edsIndex uint64, squareSize uint64) uint64 {
	shareIndex := edsIndex
	if edsIndex > squareSize {
		shareIndex = (edsIndex-(edsIndex%squareSize))/2 + (edsIndex % squareSize)
	}
	return shareIndex
}
