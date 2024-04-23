package keeper

import (
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
		prepareProposalHandler: defaultProposalHandler.PrepareProposalHandler(),
		processProposalHandler: defaultProposalHandler.ProcessProposalHandler(),
	}
}

func (h *ProofOfBlobProposalHandler) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	injectTxs := h.relayer.Reconcile(ctx)

	if len(injectTxs) > 0 {
		req.Txs = append(injectTxs, req.Txs...)
	}

	return h.prepareProposalHandler(ctx, req)
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return h.processProposalHandler(ctx, req)
}
