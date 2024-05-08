package keeper

import (
	"strings"

	tiablob "github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/lightclients/celestia"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a genesis state.
func (k *Keeper) InitGenesis(ctx sdk.Context, data *tiablob.GenesisState) error {
	for _, v := range data.Validators {
		if err := k.SetValidatorCelestiaAddress(ctx, v); err != nil {
			return err
		}
	}

	if err := k.SetProvenHeight(ctx, data.ProvenHeight); err != nil {
		return err
	}

	k.SetCelestiaGenesisState(ctx, data.CelestiaGenesisState)

	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *tiablob.GenesisState {
	vals, err := k.GetAllValidators(ctx)
	if err != nil {
		panic(err)
	}

	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		panic(err)
	}

	return &tiablob.GenesisState{
		Validators:           vals.Validators,
		ProvenHeight:         provenHeight,
		CelestiaGenesisState: k.GetCelestiaGenesisState(ctx),
	}
}

// GetCelestiaGenesisState exports the celestia light client's full state
func (k Keeper) GetCelestiaGenesisState(ctx sdk.Context) *celestia.GenesisState {
	var celestiaGenesisState celestia.GenesisState

	clientState, found := k.GetClientState(ctx)
	if found {
		var consensusStates []*celestia.ConsensusStateWithHeight
		var metadatas []*celestia.GenesisMetadata

		store := ctx.KVStore(k.storeKey)

		// Iterate through consensusStates prefix store for consensus states and metadata
		iterator := storetypes.KVStorePrefixIterator(store, []byte(celestia.KeyConsensusStatePrefix))
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			split := strings.Split(string(iterator.Key()), "/")
			// consensus state key is in the format "consensusStates/<height>"
			if len(split) == 2 {
				height := celestia.MustParseHeight(split[1])
				var consensusState celestia.ConsensusState
				k.cdc.MustUnmarshal(iterator.Value(), &consensusState)
				consensusStateWithHeight := celestia.ConsensusStateWithHeight{
					Height:         height,
					ConsensusState: consensusState,
				}
				consensusStates = append(consensusStates, &consensusStateWithHeight)
			} else {
				metadatas = append(metadatas, &celestia.GenesisMetadata{
					Key:   iterator.Key(),
					Value: iterator.Value(),
				})
			}
		}

		// Iterate through iterateConsensusStates prefix store for ordered heights metadata
		iterator2 := storetypes.KVStorePrefixIterator(store, []byte(celestia.KeyIterateConsensusStatePrefix))
		defer iterator2.Close()
		for ; iterator2.Valid(); iterator2.Next() {
			metadatas = append(metadatas, &celestia.GenesisMetadata{
				Key:   iterator2.Key(),
				Value: iterator2.Value(),
			})
		}

		celestiaGenesisState.ClientState = *clientState
		celestiaGenesisState.ConsensusStates = consensusStates
		celestiaGenesisState.Metadata = metadatas
	}

	return &celestiaGenesisState
}

// SetCelestiaGenesisState imports celestia light client's full state
func (k Keeper) SetCelestiaGenesisState(ctx sdk.Context, gs *celestia.GenesisState) {
	if gs != nil {
		store := ctx.KVStore(k.storeKey)
		for _, metadata := range gs.Metadata {
			store.Set(metadata.Key, metadata.Value)
		}
		celestia.SetClientState(store, k.cdc, &gs.ClientState)
		for _, consensusStateWithHeight := range gs.ConsensusStates {
			celestia.SetConsensusState(store, k.cdc, &consensusStateWithHeight.ConsensusState, consensusStateWithHeight.Height)
		}
	}
}
