package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (k *Keeper) PreblockerCreateClient(ctx sdk.Context, createClient *celestia.CreateClient) error {
	if createClient != nil {
		if err := k.CreateClient(ctx, createClient.ClientState, createClient.ConsensusState); err != nil {
			return fmt.Errorf("preblocker create client, %v", err)
		}
	}
	return nil
}

func (k *Keeper) PreblockerHeaders(ctx sdk.Context, headers []*celestia.Header) error {
	if headers != nil {
		for _, header := range headers {
			if err := k.UpdateClient(ctx, header); err != nil {
				return fmt.Errorf("preblocker headers, update client, %v", err)
			}
		}
	}
	return nil
}

func (k *Keeper) PreblockerProofs(ctx sdk.Context, proofs []*celestia.BlobProof) error {
	if proofs != nil {
		defer k.NotifyProvenHeight(ctx)
		for _, proof := range proofs {
			provenHeight, err := k.GetProvenHeight(ctx)
			if err != nil {
				return fmt.Errorf("preblocker proofs, get proven height, %v", err)
			}
			if proof.RollchainHeight != provenHeight+1 {
				return fmt.Errorf("preblocker proofs,  expected height: %d, actual height: %d", provenHeight+1, proof.RollchainHeight)
			}

			// Form blob
			// State sync will need to sync from a snapshot + the unproven blocks
			block, err := k.relayer.GetLocalBlockAtHeight(ctx, int64(proof.RollchainHeight))
			if err != nil {
				return fmt.Errorf("preblocker proofs, get local block at height: %d, %v", proof.RollchainHeight, err)
			}

			blockProto, err := block.Block.ToProto()
			if err != nil {
				return fmt.Errorf("preblocker proofs, block to proto, %v", err)
			}

			blockProtoBz, err := blockProto.Marshal()
			if err != nil {
				return fmt.Errorf("preblocker proofs, block proto marshal, %v", err)
			}

			// Replace blob data with our data for proof verification, do this before convert
			proof.Blob.Data = blockProtoBz
			mBlob, err := celestia.BlobFromProto(&proof.Blob)
			if err != nil {
				return fmt.Errorf("preblocker proofs, blob from proto, %v", err)
			}

			// Populate proof with data
			proof.ShareProof.Data, err = blob.BlobsToShares(mBlob)
			if err != nil {
				return fmt.Errorf("preblocker proofs, blobs to shares, %v", err)
			}

			// the update state/client was not provided, try for existing consensus state
			err = k.VerifyMembership(ctx, proof.CelestiaHeight, &proof.ShareProof)
			if err != nil {
				return fmt.Errorf("preblocker proofs, verify membership, %v", err)
			}
			if err = k.SetProvenHeight(ctx, proof.RollchainHeight); err != nil {
				return fmt.Errorf("preblocker proofs, set proven height, %v", err)
			}
		}
	}
	return nil
}

func (k *Keeper) NotifyProvenHeight(ctx sdk.Context) {
	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		fmt.Println("unable to get proven height", err)
		return
	}

	k.relayer.NotifyProvenHeight(provenHeight)
}
