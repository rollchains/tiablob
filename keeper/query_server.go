package keeper

import (
	"context"

	"github.com/rollchains/tiablob"
)

var _ tiablob.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k *Keeper) tiablob.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k *Keeper
}

// PendingValidators returns the pending validators.
func (qs queryServer) Validators(ctx context.Context, _ *tiablob.QueryValidatorsRequest) (*tiablob.QueryValidatorsResponse, error) {
	vals, err := qs.k.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	return &tiablob.QueryValidatorsResponse{Validators: vals.Validators}, nil
}

func (qs queryServer) CelestiaAddress(ctx context.Context, req *tiablob.QueryCelestiaAddressRequest) (*tiablob.QueryCelestiaAddressResponse, error) {
	addr, err := qs.k.GetValidatorCelestiaAddress(ctx, req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	return &tiablob.QueryCelestiaAddressResponse{CelestiaAddress: addr}, nil
}
