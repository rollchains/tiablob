package relayer

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	blocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/rollchains/tiablob/celestia-node/share"
	"github.com/rollchains/tiablob/light-clients/celestia"
	"github.com/rollchains/tiablob/celestia-node/blob"
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
			r.consensusStateCache[queryHeight] = r.GetLightClientHeader(ctx, queryHeight)
		}
		for _, blob := range blobs {
			proof, blockHeight, err := r.GetBlobProof(ctx, celestiaNodeClient, blob, queryHeight)
			if err != nil {
				return
			}
			if blockHeight > 0 {
				r.blockProofCache[blockHeight] = proof
			}
		}
		r.celestiaLastQueriedHeight++
	}

	// prune old blobs and consensus states
}

// Returns nil, nil if blob in namespace is not ours
// Returns an error if we need to quit
func (r Relayer) GetBlobProof(ctx context.Context, celestiaNodeClient *cn.Client, blob *blob.Blob, queryHeight uint64) (*Proof, uint64, error) {
	// Check that the blob matches our block
	var blobBlock blocktypes.Block
	err := blobBlock.Unmarshal(blob.GetData())
	if err != nil {
		r.logger.Error(fmt.Sprintf("Error unmatshalling blob data at height: %d, %v", queryHeight, err))
		return nil, 0, nil
	}

	// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
	// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
	// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
	// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
	expectedBlock, err := r.provider.GetLocalBlockAtHeight(ctx, blobBlock.Header.Height)
	if err != nil {
		r.logger.Error("Error getting block at celestia height:", queryHeight, "chain height", blobBlock.Header.Height, "err", err)
		return nil, 0, nil
	}
	// Just verify hashes for now, but expand this, maybe checksum it?
	if !reflect.DeepEqual(blobBlock.Header.AppHash, expectedBlock.Block.Header.AppHash) {
		r.logger.Error("App hash mismatch")
		return nil, 0, nil
	}
	// Get proof
	// TODO: should we get the proof at the latest height?
	proof, err := celestiaNodeClient.Blob.GetProof(ctx, queryHeight, r.celestiaNamespace.Bytes(), blob.Commitment)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Error on GetProof at height: %d, %v", queryHeight, err))
		return nil, 0, err 
	}
	included, err := celestiaNodeClient.Blob.Included(ctx, queryHeight, r.celestiaNamespace.Bytes(), proof, blob.Commitment)
	if err != nil {
		r.logger.Error("Err on included")
		return nil, 0, err
	}
	if !included {
		r.logger.Error("Not included")
		return nil, 0, fmt.Errorf("Blob not included")
	}

	fmt.Println("Proof added for height: ", blobBlock.Header.Height)
	return &Proof{
		Proof: proof,
		Height: queryHeight,
		Commitment: &blob.Commitment,
	}, uint64(blobBlock.Header.Height), nil
}

func (r Relayer) GetLightClientHeader(ctx context.Context, queryHeight uint64) *celestia.Header {
	newLightBlock, err := r.provider.QueryLightBlock(ctx, int64(queryHeight))
	if err != nil {
		r.logger.Error("error querying light block for proofs, height:", queryHeight)
		return nil
	}

	var trustedHeight celestia.Height
	if r.latestClientState != nil {
		trustedHeight = r.latestClientState.LatestHeight
	} else {
		trustedHeight = celestia.NewHeight(1, 1)
	}
	if r.latestClientState == nil {
		r.logger.Error("Client state is not set")
		return nil
	}

	trustedValidatorsInBlock, err := r.provider.QueryLightBlock(ctx, int64(trustedHeight.GetRevisionHeight()+1))
	if err != nil {
		r.logger.Error("error querying trusted light block, height:", trustedHeight)
		return nil
	}

	valSet, err := newLightBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error new light block to val set proto")
		return nil
	}

	trustedValSet, err := trustedValidatorsInBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error trusted validators in block to val set proto")
		return nil
	}

	fmt.Println("Adding light client header, trustedHeight:", trustedHeight, "new height:", newLightBlock.SignedHeader.Height)
	return &celestia.Header{
		SignedHeader: newLightBlock.SignedHeader.ToProto(),
		ValidatorSet: valSet,
		TrustedHeight: trustedHeight,
		TrustedValidators: trustedValSet,
	}
}