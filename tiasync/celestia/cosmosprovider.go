package celestia

import (
	"time"
	
	celestiarpc "github.com/tendermint/tendermint/rpc/client/http"
)

type CosmosProvider struct {
	rpcClient     *celestiarpc.HTTP
}
// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider(rpcURL string, timeout time.Duration) (*CosmosProvider, error) {
	rpcClient, err := celestiarpc.NewWithTimeout(rpcURL, "/websocket", uint(timeout.Seconds()))
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		rpcClient:      rpcClient,
	}

	return cp, nil
}
