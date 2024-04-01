package cosmos

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
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

// QueryChainID queries the chain ID from the RPC client
func (cc *CosmosProvider) QueryGranterHasGranteeForClaimAndDelegate(ctx context.Context, granter string, grantee string) (bool, bool, error) {
	res, err := authz.NewQueryClient(cc).GranteeGrants(ctx, &authz.QueryGranteeGrantsRequest{
		Grantee:    grantee,
		Pagination: defaultPageRequest(),
	})

	if err != nil {
		return false, false, fmt.Errorf("unable to query grantee grants: %w", err)
	}

	var foundClaimGrant, foundDelegateGrant bool
	for _, grant := range res.Grants {
		if grant.Granter == granter && grant.Grantee == grantee {
			if grant.Authorization.TypeUrl == "/cosmos.staking.v1beta1.MsgDelegate" {
				foundDelegateGrant = true
				if foundClaimGrant {
					break
				}
			}
			if grant.Authorization.TypeUrl == "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward" {
				foundClaimGrant = true
				if foundDelegateGrant {
					break
				}
			}
		}
	}

	return foundClaimGrant, foundDelegateGrant, nil
}
