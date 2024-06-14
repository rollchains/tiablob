package trustedrpc

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (cc *CosmosProvider) Validators(ctx context.Context, height int64) (*coretypes.ResultValidators, error) {
	vals, err := cc.rpcClient.Validators(ctx, &height, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error querying validators at latest uncommitted height")
	}
	return vals, nil
}

// Status queries the status of this node, can be used to check if it is catching up or a validator
func (cc *CosmosProvider) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying status: %w", err)
	}
	return status, nil
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
