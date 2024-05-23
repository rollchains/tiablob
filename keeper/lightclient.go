package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

var (
	keyClientStore = []byte("client_store")
)

func (k Keeper) GetClientState(ctx sdk.Context) (*celestia.ClientState, bool) {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	return celestia.GetClientState(clientStore, k.cdc)
}

func (k Keeper) CreateClient(ctx sdk.Context, clientState celestia.ClientState, consensusState celestia.ConsensusState) error {
	if err := clientState.Validate(); err != nil {
		return err
	}

	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	if err := clientState.Initialize(ctx, k.cdc, clientStore, &consensusState); err != nil {
		return err
	}

	if status := clientState.Status(ctx, clientStore, k.cdc); status != celestia.Active {
		return fmt.Errorf("client not active, cannot create client with status %s", status)
	}

	k.relayer.SetLatestClientState(&clientState)

	return nil
}

func (k Keeper) CanUpdateClient(ctx sdk.Context, clientMsg celestia.ClientMessage) error {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	clientState, found := celestia.GetClientState(clientStore, k.cdc)
	if !found {
		return fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore, k.cdc); status != celestia.Active {
		return fmt.Errorf("client not active, cannot update client with status %s", status)
	}

	if err := clientState.VerifyClientMessage(ctx, k.cdc, clientStore, clientMsg); err != nil {
		return err
	}

	return nil
}

func (k Keeper) UpdateClient(ctx sdk.Context, clientMsg celestia.ClientMessage) error {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	clientState, found := celestia.GetClientState(clientStore, k.cdc)
	if !found {
		return fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore, k.cdc); status != celestia.Active {
		return fmt.Errorf("client not active, cannot update client with status %s", status)
	}

	if err := clientState.VerifyClientMessage(ctx, k.cdc, clientStore, clientMsg); err != nil {
		return err
	}

	foundMisbehaviour := clientState.CheckForMisbehaviour(ctx, k.cdc, clientStore, clientMsg)
	if foundMisbehaviour {
		clientState.UpdateStateOnMisbehaviour(ctx, k.cdc, clientStore, clientMsg)
		return nil
	}

	_ = clientState.UpdateState(ctx, k.cdc, clientStore, clientMsg)

	k.relayer.SetLatestClientState(clientState)
	k.relayer.ClearUpdateClient()

	return nil
}

func (k Keeper) VerifyMembership(ctx sdk.Context, height uint64, proof *celestia.ShareProof) error {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	clientState, found := celestia.GetClientState(clientStore, k.cdc)
	if !found {
		return fmt.Errorf("client state not found in update client")
	}

	if status := clientState.Status(ctx, clientStore, k.cdc); status != celestia.Active {
		return fmt.Errorf("client not active, cannot update client with status %s", status)
	}

	cHeight := celestia.NewHeight(celestia.ParseChainID(clientState.ChainId), height)
	err := clientState.VerifyMembership(ctx, clientStore, k.cdc, cHeight, proof)
	if err != nil {
		return err
	}

	return nil
}
