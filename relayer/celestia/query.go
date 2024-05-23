package celestia

import (
	"context"
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/tendermint/tendermint/abci/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	types "github.com/tendermint/tendermint/types"
)

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

func (cc *CosmosProvider) ProveShares(ctx context.Context, height uint64, startShare uint64, endShare uint64) (*types.ShareProof, error) {
	res, err := cc.rpcClient.ProveShares(ctx, height, startShare, endShare)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
