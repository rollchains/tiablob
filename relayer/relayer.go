package relayer

import (
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/lightclients/celestia"
	celestiaprovider "github.com/rollchains/tiablob/relayer/celestia"
	"github.com/rollchains/tiablob/relayer/local"

	"github.com/cosmos/cosmos-sdk/codec"
)

const (
	DefaultMaxFlushSize = int(20)
	MaxMaxFlushSize     = int(100)
)

// Relayer is responsible for posting new blocks to Celestia and relaying block proofs from Celestia via the current proposer
type Relayer struct {
	logger log.Logger

	provenHeights      chan int64
	latestProvenHeight int64

	commitHeights      chan int64
	latestCommitHeight int64

	pollInterval         time.Duration
	blockProofCacheLimit int
	nodeRpcUrl           string
	nodeAuthToken        string

	// These items are shared state, must be access with mutex
	blockProofCache     map[int64]*celestia.BlobProof
	celestiaHeaderCache map[int64]*celestia.Header
	updateClient        *celestia.Header
	mu                  sync.Mutex

	celestiaProvider *celestiaprovider.CosmosProvider
	localProvider    *local.CosmosProvider
	clientCtx        client.Context

	celestiaChainID              string
	celestiaNamespace            appns.Namespace
	celestiaGasPrice             string
	celestiaGasAdjustment        float64
	celestiaPublishBlockInterval int
	celestiaLastQueriedHeight    int64
}

// NewRelayer creates a new Relayer instance
func NewRelayer(
	logger log.Logger,
	cdc codec.BinaryCodec,
	appOpts servertypes.AppOptions,
	celestiaNamespace appns.Namespace,
	keyDir string,
	celestiaPublishBlockInterval int,
) (*Relayer, error) {
	cfg := CelestiaConfigFromAppOpts(appOpts)

	if cfg.MaxFlushSize < 1 || cfg.MaxFlushSize > MaxMaxFlushSize {
		cfg.MaxFlushSize = DefaultMaxFlushSize
		//panic(fmt.Sprintf("invalid relayer max flush size: %d", cfg.MaxFlushSize))
	}

	celestiaProvider, err := celestiaprovider.NewProvider(cfg.AppRpcURL, keyDir, cfg.AppRpcTimeout, cfg.ChainID)
	if err != nil {
		return nil, err
	}

	localProvider, err := local.NewProvider(cdc)
	if err != nil {
		return nil, err
	}

	if cfg.OverrideNamespace != "" {
		celestiaNamespace = appns.MustNewV0([]byte(cfg.OverrideNamespace))
	}

	if cfg.OverridePubInterval > 0 {
		celestiaPublishBlockInterval = cfg.OverridePubInterval
	}

	return &Relayer{
		logger: logger,

		pollInterval: cfg.ProofQueryInterval,

		provenHeights: make(chan int64, 10000),
		commitHeights: make(chan int64, 10000),

		celestiaProvider:             celestiaProvider,
		localProvider:                localProvider,
		celestiaNamespace:            celestiaNamespace,
		celestiaChainID:              cfg.ChainID,
		celestiaGasPrice:             cfg.GasPrice,
		celestiaGasAdjustment:        cfg.GasAdjustment,
		celestiaPublishBlockInterval: celestiaPublishBlockInterval,
		celestiaLastQueriedHeight:    0, // Defaults to 0, but init genesis can set this based on client state's latest height

		nodeRpcUrl:    cfg.NodeRpcURL,
		nodeAuthToken: cfg.NodeAuthToken,

		blockProofCache:      make(map[int64]*celestia.BlobProof),
		blockProofCacheLimit: cfg.MaxFlushSize,
		celestiaHeaderCache:  make(map[int64]*celestia.Header),
	}, nil
}

func (r *Relayer) SetClientContext(clientCtx client.Context) {
	r.clientCtx = clientCtx
}
