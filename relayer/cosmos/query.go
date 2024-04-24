package cosmos

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func defaultPageRequest() *querytypes.PageRequest {
	return &querytypes.PageRequest{
		Key:        []byte(""),
		Offset:     0,
		Limit:      1000,
		CountTotal: false,
	}
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

// QueryLatestHeight queries the latest height from the RPC client
func (cc *CosmosProvider) QueryLatestHeight(ctx context.Context) (int64, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// GetBlockAtHeight queries the block at a given height
func (cc *CosmosProvider) GetBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	block, err := cc.rpcClient.Block(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("error querying block at height %d: %w", height, err)
	}
	return block, nil
}

// GetBlockAtHeight queries the block at a given height
func (cc *CosmosProvider) GetLocalBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	block, err := cc.localRpcClient.Block(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("error querying block at height %d: %w", height, err)
	}
	return block, nil
}

func (cc *CosmosProvider) AccountInfo(ctx context.Context, address string) (authtypes.AccountI, error) {
	res, err := authtypes.NewQueryClient(cc).Account(ctx, &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to query account: %w", err)
	}
	var acc authtypes.AccountI
	if err := cc.cdc.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return nil, fmt.Errorf("unable to unpack account: %w", err)
	}

	return acc, nil
}

// QueryChainID queries the chain ID from the RPC client
func (cc *CosmosProvider) QueryChainID(ctx context.Context) (string, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return "", err
	}
	return status.NodeInfo.Network, nil
}
