package keeper

import (
	"fmt"
	"reflect"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/rollchains/tiablob"
)

type ProofOfBlobProposalHandler struct {
	keeper *Keeper

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
	h.keeper.proposerAddress = req.ProposerAddress

	resp, err := h.prepareProposalHandler(ctx, req)
	if err != nil {
		return nil, err
	}

	latestProvenHeight, err := h.keeper.GetProvenHeight(ctx)
	if err != nil {
		return nil, err
	}

	injectData := h.keeper.prepareInjectData(ctx, req.Time, latestProvenHeight, req.ProposerAddress)
	injectDataBz := h.keeper.marshalMaxBytes(&injectData, req.MaxTxBytes, latestProvenHeight)
	resp.Txs = h.keeper.addTiablobDataToTxs(injectDataBz, req.MaxTxBytes, resp.Txs)

	return resp, nil
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedData := h.keeper.getInjectedData(req.Txs)
	if injectedData != nil {
		fmt.Println("In process proposal")
		req.Txs = req.Txs[1:] // Pop the injected data for the default handler
		if err := h.keeper.processCreateClient(ctx, injectedData.CreateClient); err != nil {
			return nil, err
		}
		if err := h.keeper.processHeaders(ctx, injectedData.Headers); err != nil {
			return nil, err
		}
		if err := h.keeper.processProofs(ctx, injectedData.Headers, injectedData.Proofs); err != nil {
			return nil, err
		}
		if err := h.keeper.processPendingBlocks(ctx, req.Time, &injectedData.PendingBlocks); err != nil {
			return nil, err
		}
		if !injectedData.BlockTime.Equal(req.Time) {
			return nil, fmt.Errorf("process proposal, expected time: %s, actual time: %s", req.Time, injectedData.BlockTime)
		}
		if !reflect.DeepEqual(injectedData.ProposerAddress, req.ProposerAddress) {
			return nil, fmt.Errorf("process proposal, expected proposer: %s, actual proposer: %s", req.ProposerAddress, injectedData.ProposerAddress)
		}
		fmt.Println("Done processing proposal")
	}
	return h.processProposalHandler(ctx, req)
}

/*func (k *Keeper) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) error {
	injectedData := k.getInjectedData(req.Txs)
	if injectedData != nil {
		if err := k.preblockerCreateClient(ctx, injectedData.CreateClient); err != nil {
			return err
		}
		if err := k.preblockerHeaders(ctx, injectedData.Headers); err != nil {
			return err
		}
		if err := k.preblockerProofs(ctx, injectedData.Proofs); err != nil {
			return err
		}
		if err := k.preblockerPendingBlocks(ctx, req.Time, req.ProposerAddress, &injectedData.PendingBlocks); err != nil {
			return err
		}
	}
	return nil
}*/

func (k *Keeper) getInjectedData(txs [][]byte) *tiablob.MsgInjectedData {
	if len(txs) != 0 {
		var injectedTx tiablob.InjectTx
		err := k.cdc.Unmarshal(txs[0], &injectedTx)
		if err == nil {
			err = sdktx.UnpackInterfaces(k.cdc.InterfaceRegistry(), injectedTx.Messages)
			if err == nil {
				msgs, err := sdktx.GetMsgs(injectedTx.Messages, "tiablob.InjectTx")
				if err == nil {
					return msgs[0].(*tiablob.MsgInjectedData)
				}
			}
		}
	}
	return nil
}
