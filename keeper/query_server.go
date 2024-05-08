package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"

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

func (qs queryServer) ProvenHeight(ctx context.Context, _ *tiablob.QueryProvenHeightRequest) (*tiablob.QueryProvenHeightResponse, error) {
	provenHeight, err := qs.k.GetProvenHeight(ctx)
	if err != nil {
		return nil, err
	}
	return &tiablob.QueryProvenHeightResponse{
		ProvenHeight: provenHeight,
	}, nil
}

func (qs queryServer) PendingBlocks(ctx context.Context, _ *tiablob.QueryPendingBlocksRequest) (*tiablob.QueryPendingBlocksResponse, error) {
	pendingBlocks, err := qs.k.GetPendingBlocksWithExpiration(ctx)
	if err != nil {
		return nil, err
	}
	return &tiablob.QueryPendingBlocksResponse{
		PendingBlocks: pendingBlocks,
	}, nil
}

func (qs queryServer) ExpiredBlocks(ctx context.Context, _ *tiablob.QueryExpiredBlocksRequest) (*tiablob.QueryExpiredBlocksResponse, error) {
	currentTimeNs := time.Now().UnixNano()
	iterator, err := qs.k.TimeoutsToPendingBlocks.
		Iterate(ctx, (&collections.Range[int64]{}).StartInclusive(0).EndInclusive(currentTimeNs))
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	var expiredBlocks []*tiablob.BlockWithExpiration
	for ; iterator.Valid(); iterator.Next() {
		expiration, err := iterator.Key()
		if err != nil {
			return nil, err
		}
		blocks, err := iterator.Value()
		if err != nil {
			return nil, err
		}
		for _, block := range blocks.BlockHeights {
			expiredBlocks = append(expiredBlocks, &tiablob.BlockWithExpiration{
				Height: block,
				Expiration: time.Unix(0, expiration),
			})
		}
	}
	return &tiablob.QueryExpiredBlocksResponse{
		CurrentTime: time.Unix(0, currentTimeNs),
		ExpiredBlocks: expiredBlocks,
	}, nil
}