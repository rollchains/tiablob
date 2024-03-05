package keeper

import (
	"context"

	"github.com/rollchains/tiablob"
)

func (k *Keeper) SetValidatorCelestiaAddress(ctx context.Context, validator tiablob.Validator) error {
	return k.Validators.Set(ctx, validator.ValidatorAddress, validator.CelestiaAddress)
}

func (k *Keeper) GetValidatorCelestiaAddress(ctx context.Context, validatorAddress string) (string, error) {
	return k.Validators.Get(ctx, validatorAddress)
}

func (k *Keeper) GetAllValidators(ctx context.Context) (tiablob.Validators, error) {
	var validators tiablob.Validators
	it, err := k.Validators.Iterate(ctx, nil)
	if err != nil {
		return validators, err
	}

	defer it.Close()

	for ; it.Valid(); it.Next() {
		var validator tiablob.Validator
		validator.ValidatorAddress, err = it.Key()
		if err != nil {
			return validators, err
		}
		validator.CelestiaAddress, err = it.Value()
		if err != nil {
			return validators, err
		}
		validators.Validators = append(validators.Validators, validator)
	}

	return validators, nil
}

func (k *Keeper) SetProvenHeight(ctx context.Context, height uint64) error {
	return k.ProvenHeight.Set(ctx, height)
}

func (k *Keeper) GetProvenHeight(ctx context.Context) (uint64, error) {
	return k.ProvenHeight.Get(ctx)
}

func (k *Keeper) SetClientID(ctx context.Context, clientID string) error {
	return k.ClientID.Set(ctx, clientID)
}

func (k *Keeper) GetClientID(ctx context.Context) (string, error) {
	return k.ClientID.Get(ctx)
}
