package cosmos

import (
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"

	client "github.com/strangelove-ventures/cometbft-client/client"
)

var _ gogogrpc.ClientConn = &CosmosProvider{}

var protoCodec = encoding.GetCodec(proto.Name)

type CosmosProvider struct {
	cdc       Codec
	rpcClient RPCClient
	keybase   keyring.Keyring

	keyDir string

	broadcastMu sync.Mutex
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider(rpcURL string, keyDir string) (*CosmosProvider, error) {
	rpcClient, err := client.NewClient(rpcURL, 5*time.Second)
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		cdc:       makeCodec(ModuleBasics),
		rpcClient: NewRPCClient(rpcClient),
		keyDir:    keyDir,
	}

	return cp, nil
}
