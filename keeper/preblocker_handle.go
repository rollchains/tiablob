package keeper

import (
	"fmt"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (k *Keeper) preblockerCreateClient(ctx sdk.Context, createClient *celestia.CreateClient) error {
	if createClient != nil {
		if err := k.CreateClient(ctx, createClient.ClientState, createClient.ConsensusState); err != nil {
			return fmt.Errorf("preblocker create client, %v", err)
		}
	}
	return nil
}

func (k *Keeper) preblockerHeaders(ctx sdk.Context, headers []*celestia.Header) error {
	if len(headers) > 0 {
		for _, header := range headers {
			if err := k.UpdateClient(ctx, header); err != nil {
				return fmt.Errorf("preblocker headers, update client, %v", err)
			}
		}
	}
	return nil
}

func (k *Keeper) preblockerProofs(ctx sdk.Context, proofs []*celestia.BlobProof) error {
	if len(proofs) > 0 {
		defer k.notifyProvenHeight(ctx)
		for _, proof := range proofs {
			provenHeight, err := k.GetProvenHeight(ctx)
			if err != nil {
				return fmt.Errorf("preblocker proofs, get proven height, %v", err)
			}
			checkHeight := provenHeight + 1
			blobs := make([]*blob.Blob, len(proof.RollchainHeights))
			for i, height := range proof.RollchainHeights {
				if height != checkHeight {
					return fmt.Errorf("preblocker proofs,  expected height: %d, actual height: %d", checkHeight, height)
				}
				checkHeight++

				// Form blob
				// State sync will need to sync from a snapshot + the unproven blocks
				blockProtoBz, err := k.relayer.GetLocalBlockAtHeight(ctx, height)
				if err != nil {
					// Check for cached unprovenBlocks
					blockProtoBz = k.unprovenBlocks[height]
					if blockProtoBz == nil {
						return fmt.Errorf("preblocker proofs, get local block at height: %d, %v", height, err)
					}
				}

				// create blob from local data
				blobs[i], err = blob.NewBlobV0(k.celestiaNamespace, blockProtoBz)
				if err != nil {
					return fmt.Errorf("preblocker proofs, blob from proto, %v", err)
				}
			}

			// Populate proof with data
			proof.ShareProof.Data, err = blob.BlobsToShares(blobs...)
			if err != nil {
				return fmt.Errorf("preblocker proofs, blobs to shares, %v", err)
			}

			// the update state/client was not provided, try for existing consensus state
			err = k.VerifyMembership(ctx, uint64(proof.CelestiaHeight), &proof.ShareProof)
			if err != nil {
				return fmt.Errorf("preblocker proofs, verify membership, %v", err)
			}
			if err = k.SetProvenHeight(ctx, proof.RollchainHeights[len(proof.RollchainHeights)-1]); err != nil {
				return fmt.Errorf("preblocker proofs, set proven height, %v", err)
			}
			for _, height := range proof.RollchainHeights {
				if err = k.RemovePendingBlock(ctx, height); err != nil {
					return fmt.Errorf("preblocker proofs, remove pending block, %v", err)
				}
			}
		}
	}
	return nil
}

func (k *Keeper) preblockerPendingBlocks(ctx sdk.Context, blockTime time.Time, proposerAddr []byte, pendingBlocks *PendingBlocks) error {
	if pendingBlocks != nil {
		if reflect.DeepEqual(k.proposerAddress, proposerAddr) {
			k.relayer.PostBlocks(ctx, pendingBlocks.BlockHeights)
		}
		for _, pendingBlock := range pendingBlocks.BlockHeights {
			if err := k.AddUpdatePendingBlock(ctx, pendingBlock, blockTime); err != nil {
				return fmt.Errorf("preblocker pending blocks, %v", err)
			}
		}
	}

	return nil
}

func (k *Keeper) notifyProvenHeight(ctx sdk.Context) {
	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		fmt.Println("unable to get proven height", err)
		return
	}

	k.relayer.NotifyProvenHeight(provenHeight)
	k.pruneUnprovenBlocksMap(provenHeight)
}

func (k *Keeper) pruneUnprovenBlocksMap(provenHeight int64) {
	for h := range k.unprovenBlocks {
		if h <= provenHeight {
			delete(k.unprovenBlocks, h)
		}
	}
}