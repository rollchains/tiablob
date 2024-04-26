package keeper

import (
	"encoding/json"

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

	clientState, clientFound := h.keeper.GetClientState(ctx)
	if clientFound {
		// Don't like this, but works for now
		h.keeper.relayer.SetLatestClientState(clientState)
	}
	injectData := h.relayer.Reconcile(ctx, clientFound)

	if !injectData.IsEmpty() {
		injectDataBz, err := json.Marshal(injectData)
		if err == nil {
			resp.Txs = append(resp.Txs, injectDataBz)
		}
	}

	return resp, err
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedData := tiablobrelayer.GetInjectedData(req.Txs)
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
	injectedData := tiablobrelayer.GetInjectedData(req.Txs)
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


