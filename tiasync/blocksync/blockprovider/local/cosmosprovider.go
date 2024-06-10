package local

import (
	cometrpc "github.com/cometbft/cometbft/rpc/client/http"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
)

type CosmosProvider struct {
	cdc       codec.BinaryCodec
	rpcClient *cometrpc.HTTP
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider() (*CosmosProvider, error) {
	interfaceRegistry := types.NewInterfaceRegistry()
	appCodec := codec.NewProtoCodec(interfaceRegistry)

	// Client wrapper seems to have issues with getting/copying the block. Validate basic does not succeed with it.
	rpcClient, err := cometrpc.NewWithTimeout("http://127.0.0.1:26657", "/websocket", uint(3))
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		cdc:       appCodec,
		rpcClient: rpcClient,
	}

	return cp, nil
}
