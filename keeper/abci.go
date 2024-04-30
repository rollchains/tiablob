package keeper

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
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

	_, clientFound := h.keeper.GetClientState(ctx)

	// TODO: return headers and proofs, get client state from above
	injectData := h.relayer.Reconcile(ctx, clientFound)

	if !injectData.IsEmpty() {
		var injectDataProto InjectedData
		injectDataProto.CreateClient = injectData.CreateClient
		injectDataProto.Headers = injectData.Headers
		injectDataProto.Proofs = injectData.Proofs
		injectDataBz, err := h.keeper.cdc.Marshal(&injectDataProto)
		if err == nil {
			resp.Txs = append(resp.Txs, injectDataBz)
		} else{
			fmt.Println("Error marshal inject data in prepare proposal")
		}
	}

	return resp, err
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedData := h.keeper.GetInjectedData(req.Txs)
	if injectedData != nil {
		req.Txs = req.Txs[:len(req.Txs)-1] // Pop the injected data for the default handler
		if err := h.keeper.ProcessCreateClient(ctx, injectedData.CreateClient); err != nil {
			return nil, err
		}
		if err := h.keeper.ProcessHeaders(ctx, injectedData.Headers); err != nil {
			return nil, err
		}
		if err := h.keeper.ProcessProofs(ctx, injectedData.Headers, injectedData.Proofs); err != nil {
			return nil, err
		}
	}
	return h.processProposalHandler(ctx, req)
}

func (k *Keeper) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) error {
	injectedData := k.GetInjectedData(req.Txs)
	if injectedData != nil {
		if err := k.PreblockerCreateClient(ctx, injectedData.CreateClient); err != nil {
			return err
		}
		if err := k.PreblockerHeaders(ctx, injectedData.Headers); err != nil {
			return err
		}
		if err := k.PreblockerProofs(ctx, injectedData.Proofs); err != nil {
			return err
		}
	}
	return nil
}

func (k *Keeper) GetInjectedData(txs [][]byte) *InjectedData {
	if len(txs) != 0 {
		var injectedData InjectedData
		err := k.cdc.Unmarshal(txs[len(txs)-1], &injectedData)
		if err == nil {
			return &injectedData
		}
	}
	return nil
}