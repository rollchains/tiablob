package local

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"

	"github.com/rollchains/tiablob"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

// GetBlockAtHeight queries the block at a given height
func (cc *CosmosProvider) GetBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	block, err := cc.rpcClient.Block(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("error querying block at height %d: %w", height, err)
	}
	return block, nil
}

// Status queries the status of this node, can be used to check if it is catching up or a validator
func (cc *CosmosProvider) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying status: %w", err)
	}
	return status, nil
}

// QueryABCI performs an ABCI query and returns the appropriate response and error sdk error code.
func (cc *CosmosProvider) QueryABCI(ctx context.Context, req abci.RequestQuery) (abci.ResponseQuery, error) {
	opts := rpcclient.ABCIQueryOptions{
		Height: req.Height,
		Prove:  req.Prove,
	}
	result, err := cc.rpcClient.ABCIQueryWithOptions(ctx, req.Path, req.Data, opts)
	if err != nil {
		return abci.ResponseQuery{}, err
	}

	if !result.Response.IsOK() {
		return abci.ResponseQuery{}, sdkErrorToGRPCError(result.Response)
	}

	return result.Response, nil
}

func (cc *CosmosProvider) QueryCelestiaClientState(ctx context.Context, height int64) (*celestia.ClientState, error) {
	key := []byte(fmt.Sprintf("%s%s", tiablob.ClientStoreKey, celestia.KeyClientState))

	req := abci.RequestQuery{
		Path: fmt.Sprintf("store/%s/key", tiablob.StoreKey),
		Height: height,
		Data: key,
		Prove: false,
	}

	res, err := cc.QueryABCI(ctx, req)
	if err != nil {
		return nil, err
	}

	var clientState celestia.ClientState
	if err = cc.cdc.Unmarshal(res.Value, &clientState); err != nil {
		return nil, err
	}

	return &clientState, nil

}

func sdkErrorToGRPCError(resp abci.ResponseQuery) error {
	switch resp.Code {
	case sdkerrors.ErrInvalidRequest.ABCICode():
		return status.Error(codes.InvalidArgument, resp.Log)
	case sdkerrors.ErrUnauthorized.ABCICode():
		return status.Error(codes.Unauthenticated, resp.Log)
	case sdkerrors.ErrKeyNotFound.ABCICode():
		return status.Error(codes.NotFound, resp.Log)
	default:
		return status.Error(codes.Unknown, resp.Log)
	}
}
