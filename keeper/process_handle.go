package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (k Keeper) processCreateClient(ctx sdk.Context, createClient *celestia.CreateClient) error {
	if createClient != nil {
		if err := k.relayer.ValidateNewClient(ctx, createClient); err != nil {
			return fmt.Errorf("invalid new client, %v", err)
		}
	}
	return nil
}

func (k Keeper) processHeaders(ctx sdk.Context, headers []*celestia.Header) error {
	if len(headers) > 0 {
		for _, header := range headers {
			if err := k.CanUpdateClient(ctx, header); err != nil {
				return fmt.Errorf("process header, can update client, %v", err)
			}
		}
	}
	return nil
}

func (k Keeper) processProofs(ctx sdk.Context, clients []*celestia.Header, proofs []*celestia.BlobProof) error {
	if len(proofs) > 0 {
		clientsMap := make(map[int64][]byte) // Celestia height to data hash/root map
		for _, client := range clients {
			clientsMap[client.SignedHeader.Header.Height] = client.SignedHeader.Header.DataHash
		}

		provenHeight, err := k.GetProvenHeight(ctx)
		if err != nil {
			return fmt.Errorf("process proofs, getting proven height, %v", err)
		}
		for _, proof := range proofs {
			if proof.RollchainHeight != provenHeight+1 {
				return fmt.Errorf("process proofs, expected height: %d, actual height: %d", provenHeight+1, proof.RollchainHeight)
			}
			// Form blob
			// State sync will need to sync from a snapshot + the unproven blocks
			block, err := k.relayer.GetLocalBlockAtHeight(ctx, int64(proof.RollchainHeight))
			if err != nil {
				return fmt.Errorf("process proofs, get local block at height: %d, %v", proof.RollchainHeight, err)
			}

			blockProto, err := block.Block.ToProto()
			if err != nil {
				return fmt.Errorf("process proofs, block to proto, %v", err)
			}

			blockProtoBz, err := blockProto.Marshal()
			if err != nil {
				return fmt.Errorf("process proofs, block proto marshal, %v", err)
			}

			// Replace blob data with our data for proof verification, do this before the convert
			proof.Blob.Data = blockProtoBz
			mBlob, err := celestia.BlobFromProto(&proof.Blob)
			if err != nil {
				return fmt.Errorf("process proofs, blob from proto, %v", err)
			}

			// Populate proof with data
			proof.ShareProof.Data, err = blob.BlobsToShares(mBlob)
			if err != nil {
				return fmt.Errorf("process proofs, blobs to shares, %v", err)
			}

			dataRoot := clientsMap[proof.CelestiaHeight]
			if dataRoot != nil {
				// We are supplying the update state/client with the proof
				shareProof, err := celestia.ShareProofFromProto(&proof.ShareProof)
				if err != nil {
					return fmt.Errorf("process proofs, shareproof from proto, %v", err)
				}

				err = shareProof.Validate(dataRoot)
				if err != nil {
					return fmt.Errorf("process proofs, shareproof validate, %v", err)
				}
			} else {
				// the update state/client was not provided, try for existing consensus state
				err := k.VerifyMembership(ctx, uint64(proof.CelestiaHeight), &proof.ShareProof)
				if err != nil {
					return fmt.Errorf("process proofs, verify membership, %v", err)
				}
			}
			provenHeight++
		}
	}
	return nil
}

func (k Keeper) processPendingBlocks(ctx sdk.Context, currentBlockTime time.Time, pendingBlocks *PendingBlocks) error {
	if pendingBlocks != nil {
		height := ctx.BlockHeight()
		numBlocks := len(pendingBlocks.BlockHeights)
		if numBlocks > 2 && numBlocks > k.publishToCelestiaBlockInterval {
			return fmt.Errorf("process pending blocks, included pending blocks (%d) exceeds limit (%d)", numBlocks, k.publishToCelestiaBlockInterval)
		}
		for _, pendingBlock := range pendingBlocks.BlockHeights {
			if pendingBlock <= 0 {
				return fmt.Errorf("process pending blocks, invalid block: %d", pendingBlock)
			}
			if pendingBlock >= height {
				return fmt.Errorf("process pending blocks, start (%d) cannot be >= this block height (%d)", pendingBlock, height)
			}
			// Check if already pending, if so, is it expired?
			if k.IsBlockPending(ctx, pendingBlock) && !k.IsBlockExpired(ctx, currentBlockTime, pendingBlock) {
				return fmt.Errorf("process pending blocks, block height (%d) is pending, but not expired", pendingBlock)
			}
			// Check if we have a proof for this block
			if k.relayer.HasCachedProof(pendingBlock) {
				return fmt.Errorf("process pending blocks, cached proof exists for block %d", pendingBlock)
			}
		}
		// Ensure publish boundries includes new blocks, once they are on-chain, they will be tracked appropriately
		newBlocks := k.relayer.ProposePostNextBlocks(ctx)
		for i, newBlock := range newBlocks {
			if newBlock != pendingBlocks.BlockHeights[i] {
				return fmt.Errorf("process pending blocks, block (%d) must be included", newBlock)
			}
		}
		// Validators do not need to check if expired pending blocks are not included. There could be a good reason for omitting them.
		// i.e. a celestia halts has been detected (backoff logic), a rollchain halt recently occurred or proposer recently restarted (relayer needs to catch up)
	}

	return nil
}
