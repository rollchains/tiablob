package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProofOfBlobProposalHandler struct {
	keeper  *Keeper

	prepareProposalHandler sdk.PrepareProposalHandler
	processProposalHandler sdk.ProcessProposalHandler
}

func NewProofOfBlobProposalHandler(
	k *Keeper,
	prepareProposalHandler sdk.PrepareProposalHandler,
	processProposalHandler sdk.ProcessProposalHandler,
) *ProofOfBlobProposalHandler {
	return &ProofOfBlobProposalHandler{
		keeper:                 k,
		prepareProposalHandler: prepareProposalHandler,
		processProposalHandler: processProposalHandler,
	}
}

func (h *ProofOfBlobProposalHandler) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	resp, err := h.prepareProposalHandler(ctx, req)
	if err != nil {
		return nil, err
	}

	var injectData InjectedData
	injectData.CreateClient = h.keeper.proposeCreateClient(ctx)
	injectData.PendingBlocks = h.keeper.proposePostBlocks(ctx, req.Time)
	injectData.Proofs = h.keeper.relayer.GetCachedProofs()
	injectData.Headers = h.keeper.relayer.GetCachedHeaders()


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

	h.keeper.relayer.SetProposerAddress(req.ProposerAddress)

	return resp, nil
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
		if err := h.keeper.ProcessPendingBlocks(ctx, req.Time, &injectedData.PendingBlocks); err != nil {
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
		if err := k.PreblockerPendingBlocks(ctx, req.Time, req.ProposerAddress, &injectedData.PendingBlocks); err != nil {
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
	if d.CreateClient != nil || len(d.Headers) != 0 || len(d.Proofs) != 0 || d.PendingBlocks.BlockHeight != nil {
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
		d.Proofs = keeper.relayer.GetCachedProofs()
		d.Headers = keeper.relayer.GetCachedHeaders()
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