package trustedrpc

import (
	"context"
	"fmt"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
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
