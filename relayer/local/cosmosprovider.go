package local

import (
	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
)

type CosmosProvider struct {
	rpcClient        *cometrpc.HTTP
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider() (*CosmosProvider, error) {
	// Client wrapper seems to have issues with getting/copying the block. Validate basic does not succeed with it.
	rpcClient, err := cometrpc.NewWithTimeout("http://127.0.0.1:26657", "/websocket", uint(3))
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		rpcClient: rpcClient,
	}

	return cp, nil
}
