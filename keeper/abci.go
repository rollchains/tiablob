package keeper

import (
	"encoding/json"
	"fmt"
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
		prepareProposalHandler: defaultProposalHandler.PrepareProposalHandler(), // Maybe pass this in
		processProposalHandler: defaultProposalHandler.ProcessProposalHandler(), // Maybe pass this in
	}
}

func (h *ProofOfBlobProposalHandler) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	resp, err := h.prepareProposalHandler(ctx, req)

	clientState, clientFound := h.keeper.GetClientState(ctx)
	fmt.Println("IsClientCreated: ", clientFound)
	if clientFound {
		// Don't like this, but works for now
		h.keeper.relayer.SetLatestClientState(clientState)
	}
	injectClientData := h.relayer.Reconcile(ctx, clientFound)

	if injectClientData.CreateClient != nil || injectClientData.UpdateClientAndProofs != nil {
		fmt.Println("prepare: injecting")
		injectClientDataBz, err := json.Marshal(injectClientData)
		if err == nil {
			fmt.Println("prepare: appending")
			resp.Txs = append(resp.Txs, injectClientDataBz)
		} else {
			fmt.Println("prepare: not appending")
		}
	} else {
		fmt.Println("prepare: not injecting")
	}

	return resp, err
}

func (h *ProofOfBlobProposalHandler) ProcessProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	injectedClientData := getInjectedClientData(req.Txs)
	if injectedClientData != nil {
		fmt.Println("process: injectedClientData != nil")
		req.Txs = req.Txs[:len(req.Txs)-1] // Pop the injected data for the default handler
		if injectedClientData.CreateClient != nil {
			fmt.Println("process: create client != nil")
			valid := h.relayer.VerifyNewClient(ctx, injectedClientData.CreateClient)
			if !valid {
				return nil, fmt.Errorf("invalid new client")
			}
		} else {
			fmt.Println("process: create client == nil")
		}
	} else {
		fmt.Println("process: injectedClientData == nil")
	}
	return h.processProposalHandler(ctx, req)
}

func (k *Keeper) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) error {
	injectedClientData := getInjectedClientData(req.Txs)
	if injectedClientData != nil {
		fmt.Println("preblocker: injectedClientData != nil")
		if injectedClientData.CreateClient != nil {
			fmt.Println("preblocker: create client != nil")
			err := k.CreateClient(ctx, injectedClientData.CreateClient.ClientState, injectedClientData.CreateClient.ConsensusState)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("preblocker: create client == nil")
		}
	} else {
		fmt.Println("preblocker: injectedClientData == nil")
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