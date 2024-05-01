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
		prepareProposalHandler: defaultProposalHandler.PrepareProposalHandler(), // Maybe pass this in
		processProposalHandler: defaultProposalHandler.ProcessProposalHandler(), // Maybe pass this in
	}
}

func (h *ProofOfBlobProposalHandler) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	resp, err := h.prepareProposalHandler(ctx, req)

	var injectData InjectedData

	// If there is no client state, create a client
	_, clientFound := h.keeper.GetClientState(ctx)
	if !clientFound {
		injectData.CreateClient = h.keeper.relayer.CreateClient(ctx)
	}

	// Call rconcile to publish next blocks (if necessary), get any cached proofs/headers, and update client if necessary
	injectData.Proofs, injectData.Headers = h.relayer.Reconcile(ctx)
	injectDataBz := injectData.MarshalMaxBytes(ctx, h.keeper, req.MaxTxBytes)
	if len(injectDataBz) > 0 {
		var txs [][]byte
		totalTxBytes := int64(len(injectDataBz))
		txs = append(txs, injectDataBz)
		for _, tx := range resp.Txs {
			totalTxBytes += int64(len(tx))
			// Append as many transactions as will fit
			if totalTxBytes <= req.MaxTxBytes {
				txs = append(txs, tx)
			} else {
				break
			}
		}
		resp.Txs = txs
	}

	return resp, err
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedData := h.keeper.GetInjectedData(req.Txs)
	if injectedData != nil {
		req.Txs = req.Txs[1:] // Pop the injected data for the default handler
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
		err := k.cdc.Unmarshal(txs[0], &injectedData)
		if err == nil {
			return &injectedData
		}
	}
	return nil
}

func (d InjectedData) IsEmpty() bool {
	if d.CreateClient != nil || len(d.Headers) != 0 || len(d.Proofs) != 0 {
		return false
	}
	return true
}

// MarshalMaxBytes will marshal the injected data ensuring it fits within the max bytes.
// If it does not fit, it will decrement the number of proofs to inject by 1 and retry.
// This new configuration will persist until the node is restarted. If a decrement is required,
// there was most likely a misconfiguration for block proof cache limit.
// Injected data is roughly 1KB/proof
func (d InjectedData) MarshalMaxBytes(ctx sdk.Context, keeper *Keeper, maxBytes int64) []byte {
	if d.IsEmpty() {
		return nil
	}

	injectDataBz, err := keeper.cdc.Marshal(&d)
	if err != nil {
		return nil
	}

	for ; int64(len(injectDataBz)) > maxBytes; {
		keeper.relayer.DecrementBlockProofCacheLimit()
		d.Proofs, d.Headers = keeper.relayer.Reconcile(ctx)
		if len(d.Proofs) == 0 {
			return nil
		}
		injectDataBz, err = keeper.cdc.Marshal(&d)
		if err != nil {
			return nil
		}
	}
	
	return injectDataBz
}