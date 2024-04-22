package keeper

import (
	"encoding/json"
	//"fmt"
	//"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"

	//"github.com/rollchains/tiablob/celestia-node/blob"
	//"github.com/rollchains/tiablob/light-clients/celestia"
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
	clientFound := h.keeper.IsClientCreated(ctx)
	injectClientData := h.relayer.Reconcile(ctx, clientFound)

	if injectClientData.CreateClient != nil || injectClientData.UpdateClientAndProofs != nil {
		injectClientDataBz, err := json.Marshal(injectClientData)
		if err == nil {
			req.Txs = append(req.Txs, injectClientDataBz)
		}
	}

	return h.prepareProposalHandler(ctx, req)
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return h.processProposalHandler(ctx, req)
}
