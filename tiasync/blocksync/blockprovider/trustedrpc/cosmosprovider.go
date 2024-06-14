package trustedrpc

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
func NewProvider(remote string) (*CosmosProvider, error) {
	interfaceRegistry := types.NewInterfaceRegistry()
	appCodec := codec.NewProtoCodec(interfaceRegistry)

	rpcClient, err := cometrpc.NewWithTimeout(remote, "/websocket", uint(10))
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		cdc:       appCodec,
		rpcClient: rpcClient,
	}

	return cp, nil
}
