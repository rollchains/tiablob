package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/rollchains/tiablob/light-clients/celestia"

	"cosmossdk.io/store/prefix"
)

var (
	keyClientStore = []byte("client_store")
)

func (k Keeper) IsClientCreated(ctx sdk.Context) bool {
	clientStore := prefix.NewStore(ctx.KVStore(k.storeKey), keyClientStore)

	_, found := celestia.GetClientState(clientStore)

	return found
}

func(k Keeper) CreateClient(ctx sdk.Context, clientStateBz, consensusStateBz []byte) error {
	var clientState celestia.ClientState
	err := json.Unmarshal(clientStateBz, &clientState)
	if err != nil {
		return fmt.Errorf("invalid client, %v", err)
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState celestia.ConsensusState
	err = json.Unmarshal(consensusStateBz, &consensusState)
	if err != nil {
		return fmt.Errorf("invalid consensus state, %v", err)
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