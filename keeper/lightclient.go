package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	celestiatypes "github.com/tendermint/tendermint/types"

	"github.com/rollchains/tiablob/light-clients/celestia"
)

var (
	keyClientStore = []byte("client_store")
)

func (k Keeper) GetClientState(ctx sdk.Context) (*celestia.ClientState, bool) {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	return celestia.GetClientState(clientStore)
}

func(k Keeper) CreateClient(ctx sdk.Context, clientState celestia.ClientState, consensusState celestia.ConsensusState) error {
	if err := clientState.Validate(); err != nil {
		return err
	}

	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	if err := clientState.Initialize(ctx, clientStore, &consensusState); err != nil {
		return err
	}

	if status := clientState.Status(ctx, clientStore); status != celestia.Active {
		return fmt.Errorf("client not active, cannot create client with status %s", status)
	}

	initialHeight := clientState.LatestHeight
	fmt.Println("Client create at height:", initialHeight)

	// TODO: temp solution for now...
	k.relayer.SetLatestClientState(&clientState)

	return nil
}

func (k Keeper) CanUpdateClient(ctx sdk.Context, clientMsg celestia.ClientMessage) (bool, error) {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)
	
	clientState, found := celestia.GetClientState(clientStore)
	if !found {
		return false, fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore); status != celestia.Active {
		return false, fmt.Errorf("client not active, cannot update client with status %s", status)
	}

	if err := clientState.VerifyClientMessage(ctx, clientStore, clientMsg); err != nil {
		return false, err
	}

	return true, nil
}

func (k Keeper) UpdateClient(ctx sdk.Context, clientMsg celestia.ClientMessage) error {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)
	
	clientState, found := celestia.GetClientState(clientStore)
	if !found {
		return fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore); status != celestia.Active {
		return fmt.Errorf("client not active, cannot update client with status %s", status)
	}

	if err := clientState.VerifyClientMessage(ctx, clientStore, clientMsg); err != nil {
		return err
	}

	foundMisbehaviour := clientState.CheckForMisbehaviour(ctx, clientStore, clientMsg)
	if foundMisbehaviour {
		clientState.UpdateStateOnMisbehaviour(ctx, clientStore, clientMsg)
		return nil
	}

	_ = clientState.UpdateState(ctx, clientStore, clientMsg)

	return nil
}

func (k Keeper) VerifyMembership(ctx sdk.Context, height uint64, proof *celestiatypes.ShareProof) error {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	clientState, found := celestia.GetClientState(clientStore)
	if !found {
		return fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore); status != celestia.Active {
		return fmt.Errorf("client not active, cannot update client with status %s", status)
	}
	
	cHeight := celestia.NewHeight(celestia.ParseChainID(clientState.ChainId), height)
	err := clientState.VerifyMembership(ctx, clientStore, cHeight, proof)
	if err != nil {
		return err
	}

	return nil
}