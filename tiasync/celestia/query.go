package celestia

import (
	"context"
)
// QueryLatestHeight queries the latest height from the RPC client
func (cc *CosmosProvider) QueryLatestHeight(ctx context.Context) (int64, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// QueryChainID queries the chain ID from the RPC client
func (cc *CosmosProvider) QueryChainID(ctx context.Context) (string, error) {
	status, err := cc.rpcClient.Status(ctx)
	if err != nil {
		return "", err
	}
	return status.NodeInfo.Network, nil
}
