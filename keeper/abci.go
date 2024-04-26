package keeper

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/light-clients/celestia"

	prototypes "github.com/cometbft/cometbft/proto/tendermint/types"
)

type ProofOfBlobProposalHandler struct {
	keeper  *Keeper
	relayer *tiablobrelayer.Relayer

	prepareProposalHandler sdk.PrepareProposalHandler
	processProposalHandler sdk.ProcessProposalHandler
}

func NewProofOfBlobProposalHandler(k *Keeper, r *tiablobrelayer.Relayer, mp mempool.Mempool, txVerifier baseapp.ProposalTxVerifier) *ProofOfBlobProposalHandler {
	defaultProposalHandler := baseapp.NewDefaultProposalHandler(mp, txVerifier)
	return &ProofOfBlobProposalHandler{
		keeper:                 k,
		relayer:                r,
		prepareProposalHandler: defaultProposalHandler.PrepareProposalHandler(), // Maybe pass this in
		processProposalHandler: defaultProposalHandler.ProcessProposalHandler(), // Maybe pass this in
	}
}

func (h *ProofOfBlobProposalHandler) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	resp, err := h.prepareProposalHandler(ctx, req)

	clientState, clientFound := h.keeper.GetClientState(ctx)
	if clientFound {
		// Don't like this, but works for now
		h.keeper.relayer.SetLatestClientState(clientState)
	}
	injectClientData := h.relayer.Reconcile(ctx, clientFound)

	if injectClientData.CreateClient != nil || len(injectClientData.Headers) != 0 || len(injectClientData.Proofs) != 0 {
		injectClientDataBz, err := json.Marshal(injectClientData)
		if err == nil {
			resp.Txs = append(resp.Txs, injectClientDataBz)
		}
	}

	return resp, err
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedClientData := getInjectedClientData(req.Txs)
	if injectedClientData != nil {
		req.Txs = req.Txs[:len(req.Txs)-1] // Pop the injected data for the default handler
		if injectedClientData.CreateClient != nil {
			valid := h.relayer.ValidateNewClient(ctx, injectedClientData.CreateClient)
			if !valid {
				return nil, fmt.Errorf("invalid new client")
			}
		}
		if injectedClientData.Headers != nil && !h.keeper.ProcessVerifyNewClients(ctx, injectedClientData.Headers) {
			fmt.Println("Invalid new clients")
		}
		if injectedClientData.Proofs != nil && !h.keeper.ProcessVerifyProofs(ctx, injectedClientData.Headers, injectedClientData.Proofs) {
			fmt.Println("Invalid proofs")
		}
	}
	return h.processProposalHandler(ctx, req)
}

func (k *Keeper) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) error {
	injectedClientData := getInjectedClientData(req.Txs)
	if injectedClientData != nil {
		if injectedClientData.CreateClient != nil {
			err := k.CreateClient(ctx, injectedClientData.CreateClient.ClientState, injectedClientData.CreateClient.ConsensusState)
			if err != nil {
				return err
			}
		}
		if injectedClientData.Headers != nil {
			if err := k.PreblockerUpdateStates(ctx, injectedClientData.Headers); err != nil {
				return err
			}
		}
		if injectedClientData.Proofs != nil {
			k.PreblockerVerifyProofs(ctx, injectedClientData.Proofs)
		}
	}
	return nil
}

func getInjectedClientData(txs [][]byte) *tiablobrelayer.InjectClientData {
	if len(txs) != 0 {
		var injectedClientData tiablobrelayer.InjectClientData
		err := json.Unmarshal(txs[len(txs)-1], &injectedClientData)
		if err == nil {
			return &injectedClientData
		}
	}
	return nil
}

func (k Keeper) ProcessVerifyNewClients(ctx sdk.Context, clients []*tiablobrelayer.Header) bool {
	for _, client := range clients {
		var valSet prototypes.ValidatorSet
		err := k.cdc.Unmarshal(client.ValidatorSet, &valSet)
		if err != nil {
			fmt.Println("Error val set unmarshal")
			return false
		}
		var trustedValSet prototypes.ValidatorSet
		err = k.cdc.Unmarshal(client.TrustedValidators, &trustedValSet)
		if err != nil {
			fmt.Println("Error trusted val set unmarshal")
			return false
		}
		client2 := celestia.Header{
			SignedHeader: client.SignedHeader,
			ValidatorSet: &valSet,
			TrustedHeight: client.TrustedHeight,
			TrustedValidators: &trustedValSet,
		}
		valid, err := k.CanUpdateClient(ctx, &client2)
		if err != nil {
			fmt.Println("Error validating new client", err)
			return false
		}
		if !valid {
			fmt.Println("Invalid client")
			return false
		}
	}
	return true
}

func (k Keeper) ProcessVerifyProofs(ctx sdk.Context, clients []*tiablobrelayer.Header, proofs []*tiablobrelayer.Proof) bool {
	clientsMap := make(map[uint64][]byte) // Celestia height to data hash/root map
	for _, client := range clients {
		clientsMap[uint64(client.SignedHeader.Header.Height)] = client.SignedHeader.Header.DataHash
	}

	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		fmt.Println("unable to get proven height", err)
		return false
	}
	// Do we need to sort proofs first?
	for _, proof := range proofs {
		if proof.RollchainHeight != provenHeight + 1 {
			fmt.Println("process: not a proof for provenHeight + 1, provenHeight+1:", provenHeight+1, "height:", proof.RollchainHeight)
			return false
		}
		// Form blob
		// State sync will need to sync from a snapshot + the unproven blocks
		block,  err := k.relayer.GetLocalBlockAtHeight(ctx, int64(proof.RollchainHeight))
		if err != nil {
			fmt.Println("Error getting block at rollchain height", proof.RollchainHeight, "err", err)
			//r.logger.Error("Error getting block at rollchain height", proof.RollchainHeight, "err", err)
			return false
		}
	
		blockProto, err := block.Block.ToProto()
		if err != nil {
			fmt.Println("error expected block to proto", err.Error())
			//r.logger.Error("error expected block to proto", err.Error())
			return false
		}
		
		blockProtoBz, err := blockProto.Marshal()
		if err != nil {
			fmt.Println("error blockProto marshal", err.Error())
			//r.logger.Error("error blockProto marshal", err.Error())
			return false
		}
	
		// Replace blob data with our data for proof verification
		proof.Blob.Data = blockProtoBz
	
		// Populate proof with data
		proof.ShareProof.Data, err = blob.BlobsToShares(proof.Blob)
		if err != nil {
			fmt.Println("error BlobsToShares")
			//r.logger.Error("error BlobsToShares")
			return false
		}

		dataRoot := clientsMap[proof.CelestiaHeight]
		if dataRoot != nil {
			// We are supplying the update state/client with the proof
			err := proof.ShareProof.Validate(dataRoot)
			if err != nil {
				fmt.Println("Process: ShareProof.Validate failed", err)
				return false
			}
		} else {
			// the update state/client was not provided, try for existing consensus state
			err := k.VerifyMembership(ctx, proof.CelestiaHeight, proof.ShareProof)
			if err != nil {
				fmt.Println("Process: VerifyMembership failed", err)
				return false
			}
		}
		provenHeight++
	}
	return true
}


func (k *Keeper) PreblockerUpdateStates(ctx sdk.Context, clients []*tiablobrelayer.Header) error {
	for _, client := range clients {
		var valSet prototypes.ValidatorSet
		err := k.cdc.Unmarshal(client.ValidatorSet, &valSet)
		if err != nil {
			fmt.Println("Error val set unmarshal")
			return err
		}
		var trustedValSet prototypes.ValidatorSet
		err = k.cdc.Unmarshal(client.TrustedValidators, &trustedValSet)
		if err != nil {
			fmt.Println("Error trusted val set unmarshal")
			return err
		}
		client2 := celestia.Header{
			SignedHeader: client.SignedHeader,
			ValidatorSet: &valSet,
			TrustedHeight: client.TrustedHeight,
			TrustedValidators: &trustedValSet,
		}
		err = k.UpdateClient(ctx, &client2)
		if err != nil {
			fmt.Println("Error updating new client", err)
			return err
		}
	}
	return nil
}

func (k *Keeper) PreblockerVerifyProofs(ctx sdk.Context, proofs []*tiablobrelayer.Proof) {
	defer k.NotifyProvenHeight(ctx)
	// Do we need to sort proofs first?
	for _, proof := range proofs {
		provenHeight, err := k.GetProvenHeight(ctx)
		if err != nil {
			fmt.Println("unable to get proven height", err)
			return
		}
		if proof.RollchainHeight != provenHeight + 1 {
			fmt.Println("preblocker: not a proof for provenHeight + 1, provenHeight+1:", provenHeight+1, "height:", proof.RollchainHeight)
			return
		}

		// Form blob
		// State sync will need to sync from a snapshot + the unproven blocks
		block,  err := k.relayer.GetLocalBlockAtHeight(ctx, int64(proof.RollchainHeight))
		if err != nil {
			fmt.Println("Error getting block at rollchain height", proof.RollchainHeight, "err", err)
			//r.logger.Error("Error getting block at rollchain height", proof.RollchainHeight, "err", err)
			return
		}
	
		blockProto, err := block.Block.ToProto()
		if err != nil {
			fmt.Println("error expected block to proto", err.Error())
			//r.logger.Error("error expected block to proto", err.Error())
			return
		}
		
		blockProtoBz, err := blockProto.Marshal()
		if err != nil {
			fmt.Println("error blockProto marshal", err.Error())
			//r.logger.Error("error blockProto marshal", err.Error())
			return
		}
	
		// Replace blob data with our data for proof verification
		proof.Blob.Data = blockProtoBz
	
		// Populate proof with data
		proof.ShareProof.Data, err = blob.BlobsToShares(proof.Blob)
		if err != nil {
			fmt.Println("error BlobsToShares")
			//r.logger.Error("error BlobsToShares")
			return
		}

		// the update state/client was not provided, try for existing consensus state
		err = k.VerifyMembership(ctx, proof.CelestiaHeight, proof.ShareProof)
		if err != nil {
			fmt.Println("Process: VerifyMembership failed", err)
			return
		}
		if err = k.SetProvenHeight(ctx, proof.RollchainHeight); err != nil {
			fmt.Println("error setting proven height: ", proof.RollchainHeight)
			return
		}
	}
}

func (k *Keeper) NotifyProvenHeight(ctx sdk.Context) {
	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		fmt.Println("unable to get proven height", err)
		return
	}

	k.relayer.NotifyProvenHeight(provenHeight)
}