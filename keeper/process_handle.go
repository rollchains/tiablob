package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/light-clients/celestia"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"

	prototypes "github.com/cometbft/cometbft/proto/tendermint/types"
)

func (k Keeper) ProcessCreateClient(ctx sdk.Context, createClient *tiablobrelayer.CreateClient) error {
	if createClient != nil {
		if err := k.relayer.ValidateNewClient(ctx, createClient); err != nil {
			return fmt.Errorf("invalid new client, %v", err)
		}
	}
	return nil
}

func (k Keeper) ProcessHeaders(ctx sdk.Context, headers []*tiablobrelayer.Header) error {
	if headers != nil {
		for _, header := range headers {
			var valSet prototypes.ValidatorSet
			if err := k.cdc.Unmarshal(header.ValidatorSet, &valSet); err != nil {
				return fmt.Errorf("process header, val set unmarshal, %v", err)
			}
			var trustedValSet prototypes.ValidatorSet
			if err := k.cdc.Unmarshal(header.TrustedValidators, &trustedValSet); err != nil {
				return fmt.Errorf("process header, trusted val set unmarshal, %v", err)
			}
			cHeader := celestia.Header{
				SignedHeader:      header.SignedHeader,
				ValidatorSet:      &valSet,
				TrustedHeight:     header.TrustedHeight,
				TrustedValidators: &trustedValSet,
			}
			
			if err := k.CanUpdateClient(ctx, &cHeader); err != nil {
				return fmt.Errorf("process header, can update client, %v", err)
			}
		}
	}
	return nil
}

func (k Keeper) ProcessProofs(ctx sdk.Context, clients []*tiablobrelayer.Header, proofs []*tiablobrelayer.Proof) error {
	if proofs != nil {
		clientsMap := make(map[uint64][]byte) // Celestia height to data hash/root map
		for _, client := range clients {
			clientsMap[uint64(client.SignedHeader.Header.Height)] = client.SignedHeader.Header.DataHash
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
	
			// Replace blob data with our data for proof verification
			proof.Blob.Data = blockProtoBz
	
			// Populate proof with data
			proof.ShareProof.Data, err = blob.BlobsToShares(proof.Blob)
			if err != nil {
				return fmt.Errorf("process proofs, blobs to shares, %v", err)
			}
	
			dataRoot := clientsMap[proof.CelestiaHeight]
			if dataRoot != nil {
				// We are supplying the update state/client with the proof
				err := proof.ShareProof.Validate(dataRoot)
				if err != nil {
					return fmt.Errorf("process proofs, shareproof validate, %v", err)
				}
			} else {
				// the update state/client was not provided, try for existing consensus state
				err := k.VerifyMembership(ctx, proof.CelestiaHeight, proof.ShareProof)
				if err != nil {
					return fmt.Errorf("process proofs, verify membership, %v", err)
				}
			}
			provenHeight++
		}	
	}
	return nil
}
