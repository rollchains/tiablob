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
		checkHeight = checkHeight + int64(len(proof.RollchainHeights))
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
			if err = r.getBlobProofs(ctx, blobs, queryHeight, squareSize, latestClientState); err != nil {
				r.logger.Error("getting blob proofs", "error", err)
			}
		}

		r.celestiaLastQueriedHeight = queryHeight
	}
}

func (r *Relayer) getBlobProofs(ctx context.Context, blobs []*blob.Blob, queryHeight int64, squareSize uint64, latestClientState *celestia.ClientState) error {
	success := r.tryGetAggregatedProof(ctx, blobs, queryHeight, squareSize, latestClientState)
	if !success {
		for _, mBlob := range blobs {
			err := r.getBlobProof(ctx, mBlob, queryHeight, squareSize, latestClientState)
			if err != nil {
				r.logger.Error("getting blob proof", "error", err)
				return err
			}
		}
	}
	return nil
}

// tryGetAggregatedProof will try and find a proof for multiple blocks/blobs
// currently, it is strict and will bail on an error or not sequential set of blocks
func (r *Relayer) tryGetAggregatedProof(ctx context.Context, blobs []*blob.Blob, queryHeight int64, squareSize uint64, latestClientState *celestia.ClientState) bool {
	heights := make([]int64, len(blobs))
	for i, mBlob := range blobs {
		var err error
		// Replace blob data with our data for proof verification
		heights[i], mBlob.Data = r.getBlobHeightAndData(ctx, mBlob, queryHeight, r.latestProvenHeight)
		if heights[i] == 0 || mBlob.Data == nil {
			r.logger.Error("aggregate, getBlobHeightAndData bail", "error", err)
			// bail immediately and try individual block proofs
			return false
		}

		// Non-sequential set of blocks check
		if i > 0 && heights[i] != heights[i-1]+1 {
			r.logger.Error("aggregate, blobs not sequential", "current", heights[i], "previous", heights[i-1])
			return false
		}
	}

	shares, err := blob.BlobsToShares(blobs...)
	if err != nil {
		r.logger.Error("aggregate, blobs to shares", "error", err)
		return false
	}

	shareIndex := getShareIndex(uint64(blobs[0].Index()), squareSize)

	// Get all shares from celestia node, confirm our shares are present
	proverShareProof, err := r.celestiaProvider.ProveShares(ctx, uint64(queryHeight), shareIndex, shareIndex+uint64(len(shares)))
	if err != nil {
		r.logger.Error("aggregate, error calling ProveShares", "error", err)
		return false
	}

	// Get the header to verify the proof
	header := r.getCachedHeader(queryHeight)
	if header == nil {
		header = r.fetchNewHeader(ctx, queryHeight, latestClientState)
		if header == nil {
			r.logger.Error("aggregate, fetch header is nil")
			return false
		}
	}

	proverShareProof.Data = shares // Replace with retrieved shares
	err = proverShareProof.Validate(header.SignedHeader.Header.DataHash)
	if err != nil {
		r.logger.Error("aggregate, failed verify membership", "error", err)
		return false
	}

	// We have a valid proof, if we haven't cached the header, cache it
	// This goroutine is the only way to delete headers, so it should be kept that way
	if r.getCachedHeader(queryHeight) == nil {
		r.setCachedHeader(queryHeight, header)
	}

	proverShareProof.Data = nil // Only include proof
	shareProofProto := celestia.TmShareProofToProto(proverShareProof)

	proof := &celestia.BlobProof{
		ShareProof:       shareProofProto,
		CelestiaHeight:   queryHeight,
		RollchainHeights: heights,
	}

	for _, height := range heights {
		r.setCachedProof(height, proof)
	}

	return true
}

// GetBlobProof
//   - unmarshals blob into block
//   - fetches same block from this node, converts to proto type, and marshals
//   - fetches proof from celestia app
//   - verifies proof using block data from this node
//   - stores proof in cache
//
// returns an error only when we shouldn't hit an error, otherwise nil if the blob could be someone elses (i.e. same namespace used)
func (r *Relayer) getBlobProof(ctx context.Context, mBlob *blob.Blob, queryHeight int64, squareSize uint64, latestClientState *celestia.ClientState) error {
	rollchainBlockHeight, data := r.getBlobHeightAndData(ctx, mBlob, queryHeight, r.latestProvenHeight)
	if data == nil {
		return nil
	}

	// Replace blob data with our data for proof verification
	mBlob.Data = data

	shares, err := blob.BlobsToShares(mBlob)
	if err != nil {
		r.logger.Error("error BlobsToShares")
		return nil
	}

	shareIndex := getShareIndex(uint64(mBlob.Index()), squareSize)

	// Get all shares from celestia node, confirm our shares are present
	proverShareProof, err := r.celestiaProvider.ProveShares(ctx, uint64(queryHeight), shareIndex, shareIndex+uint64(len(shares)))
	if err != nil {
		r.logger.Error("error calling ProveShares", "error", err)
		return nil
	}

	// Get the header to verify the proof
	header := r.getCachedHeader(queryHeight)
	if header == nil {
		header = r.fetchNewHeader(ctx, queryHeight, latestClientState)
		if header == nil {
			r.logger.Error("fetch new header is nil")
			return nil
		}
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
	shareProofProto := celestia.TmShareProofToProto(proverShareProof)

	r.setCachedProof(rollchainBlockHeight, &celestia.BlobProof{
		ShareProof:       shareProofProto,
		CelestiaHeight:   queryHeight,
		RollchainHeights: []int64{rollchainBlockHeight},
	})

	return nil
}

func (r *Relayer) getBlobHeightAndData(ctx context.Context, mBlob *blob.Blob, queryHeight int64, latestProvenHeight int64) (int64, []byte) {
	var blobBlockProto protoblocktypes.Block
	err := blobBlockProto.Unmarshal(mBlob.GetData())
	if err != nil {
		r.logger.Error("blob unmarshal", "height", queryHeight, "error", err)
		return 0, nil
	}

	rollchainBlockHeight := blobBlockProto.Header.Height

	// Ignore any blocks that are <= the latest proven height
	if rollchainBlockHeight <= latestProvenHeight {
		return 0, nil
	}

	blockProtoBz, err := r.GetLocalBlockAtHeight(ctx, rollchainBlockHeight)
	if err != nil {
		return 0, nil
	}

	return rollchainBlockHeight, blockProtoBz
}

func (r *Relayer) GetLocalBlockAtHeight(ctx context.Context, rollchainBlockHeight int64) ([]byte, error) {
	// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
	// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
	// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
	// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
	expectedBlock, err := r.localProvider.GetBlockAtHeight(ctx, rollchainBlockHeight)
	if err != nil {
		// TODO: add retries, bad if this errors
		r.logger.Error("getting local block", "rollchain height", rollchainBlockHeight, "err", err)
		return nil, err
	}

	blockProto, err := expectedBlock.Block.ToProto()
	if err != nil {
		return nil, err
	}

	return blockProto.Marshal()
}

// GetShareIndex calculates the share index given the EDS index of the blob and square size of the respective block
func getShareIndex(edsIndex uint64, squareSize uint64) uint64 {
	shareIndex := edsIndex
	if edsIndex > squareSize {
		shareIndex = (edsIndex-(edsIndex%squareSize))/2 + (edsIndex % squareSize)
	}
	return shareIndex
}
