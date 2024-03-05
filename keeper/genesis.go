package keeper

import (
	"context"

	tiablob "github.com/rollchains/tiablob"
)

// InitGenesis initializes the module's state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *tiablob.GenesisState) error {
	for _, v := range data.Validators {
		if err := k.SetValidatorCelestiaAddress(ctx, v); err != nil {
			return err
		}
	}

	if err := k.SetClientID(ctx, data.ClientId); err != nil {
		return err
	}

	if err := k.SetProvenHeight(ctx, data.ProvenHeight); err != nil {
		return err
	}

	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *tiablob.GenesisState {
	vals, err := k.GetAllValidators(ctx)
	if err != nil {
		panic(err)
	}

	clientID, err := k.GetClientID(ctx)
	if err != nil {
		panic(err)
	}

	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		panic(err)
	}

	return &tiablob.GenesisState{
		Validators:   vals.Validators,
		ClientId:     clientID,
		ProvenHeight: provenHeight,
	}
}
