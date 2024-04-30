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

// GetCachedProofs() returns the next set of proofs to verify on-chain
func (r *Relayer) GetCachedProofs() []*celestia.BlobProof {
	var proofs []*celestia.BlobProof
	for checkHeight := r.latestProvenHeight + 1; r.blockProofCache[checkHeight] != nil; checkHeight++ {
		proofs = append(proofs, r.blockProofCache[checkHeight])
	}

	return proofs
}

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal
// that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context) {
	celestiaNodeClient, err := cn.NewClient(ctx, r.nodeRpcUrl, r.nodeAuthToken)
	if err != nil {
		r.logger.Error("creating celestia node client", "error", err)
		return
	}
	defer celestiaNodeClient.Close()

	celestiaLatestHeight, err := r.provider.QueryLatestHeight(ctx)
	if err != nil {
		r.logger.Error("querying latest height from Celestia", "error", err)
		return
	}

	squareSize := uint64(0)
	for r.celestiaLastQueriedHeight < celestiaLatestHeight {
		// get the namespace blobs from that height
		queryHeight := uint64(r.celestiaLastQueriedHeight + 1)
		blobs, err := celestiaNodeClient.Blob.GetAll(ctx, queryHeight, []share.Namespace{r.celestiaNamespace.Bytes()})
		if err != nil {
			// this error just indicates we don't have a blob at this height
			if strings.Contains(err.Error(), "blob: not found") {
				r.celestiaLastQueriedHeight++
				continue
			}
			r.logger.Error("Celestia node blob getall", "height", queryHeight, "error", err)
			return
		}
		if len(blobs) > 0 {
			r.celestiaHeaderCache[queryHeight] = r.FetchNewHeader(ctx, queryHeight)
			block, err := r.provider.GetCelestiaBlockAtHeight(ctx, int64(queryHeight))
			if err != nil {
				r.logger.Error("querying celestia block", "height", queryHeight, "error", err)
				return
			}
			squareSize = block.Block.Data.SquareSize
		}
		for _, mBlob := range blobs {
			// TODO: we can aggregate adjacent blobs and prove with a single proof, fall back to individual blobs when not possible
			err = r.GetBlobProof(ctx, celestiaNodeClient, mBlob, queryHeight, squareSize)
			if err != nil {
				r.logger.Error("getting blob proof", "error", err)
				return
			}
		}
		r.celestiaLastQueriedHeight++
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
func (r *Relayer) GetBlobProof(ctx context.Context, celestiaNodeClient *cn.Client, mBlob *blob.Blob, queryHeight uint64, squareSize uint64) error {
	var blobBlockProto protoblocktypes.Block
	err := blobBlockProto.Unmarshal(mBlob.GetData())
	if err != nil {
		r.logger.Error("blob unmarshal", "height", queryHeight, "error", err)
		return nil
	}

	rollchainBlockHeight := blobBlockProto.Header.Height

	// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
	// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
	// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
	// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
	expectedBlock, err := r.provider.GetLocalBlockAtHeight(ctx, rollchainBlockHeight)
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
	proverShareProof, err := r.provider.ProveShares(ctx, queryHeight, shareIndex, shareIndex+uint64(len(shares)))
	if err != nil {
		r.logger.Error("error calling ProveShares", "error", err)
		return nil
	}

	proverShareProof.Data = shares // Replace with retrieved shares
	err = proverShareProof.Validate(r.celestiaHeaderCache[queryHeight].SignedHeader.Header.DataHash)
	if err != nil {
		r.logger.Error("failed verify membership", "error", err)
		return nil
	}

	proverShareProof.Data = nil // Only include proof
	mBlob.Data = nil            // Only include blob metadata
	shareProofProto := celestia.TmShareProofToProto(proverShareProof)

	r.blockProofCache[uint64(rollchainBlockHeight)] = &celestia.BlobProof{
		ShareProof:      shareProofProto,
		Blob:            celestia.BlobToProto(mBlob),
		CelestiaHeight:  queryHeight,
		RollchainHeight: uint64(rollchainBlockHeight),
	}

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
