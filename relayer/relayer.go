package relayer

import (
	"context"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/lightclients/celestia"
	celestiaprovider "github.com/rollchains/tiablob/relayer/celestia"
	"github.com/rollchains/tiablob/relayer/local"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

const (
	DefaultMaxFlushSize = int(100)
	MaxMaxFlushSize = int(100)
)

// Relayer is responsible for posting new blocks to Celestia and relaying block proofs from Celestia via the current proposer
type Relayer struct {
	logger log.Logger

	provenHeights      chan uint64
	latestProvenHeight uint64

	commitHeights      chan uint64
	latestCommitHeight uint64

	pollInterval time.Duration

	blockProofCache      map[uint64]*celestia.BlobProof
	blockProofCacheLimit int
	celestiaHeaderCache  map[uint64]*celestia.Header

	celestiaProvider  *celestiaprovider.CosmosProvider
	localProvider     *local.CosmosProvider
	clientCtx client.Context

	latestClientState *celestia.ClientState
	updateClient      *celestia.Header

	nodeRpcUrl    string
	nodeAuthToken string

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

	localProvider, err := local.NewProvider()

	return &Relayer{
		logger: logger,

		pollInterval: cfg.ProofQueryInterval,

		provenHeights: make(chan uint64, 10000),
		commitHeights: make(chan uint64, 10000),

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

		blockProofCache:      make(map[uint64]*celestia.BlobProof),
		blockProofCacheLimit: cfg.MaxFlushSize,
		celestiaHeaderCache:  make(map[uint64]*celestia.Header),
	}, nil
}

func (r *Relayer) SetClientContext(clientCtx client.Context) {
	r.clientCtx = clientCtx
}

// SetCelestiaLastQueriedHeight allows tiablob keeper to set a starting point for querying Celestia blocks on import genesis
// This may not be needed if we query tx hashes for blob heights
func (r *Relayer) SetCelestiaLastQueriedHeight(height int64) {
	r.celestiaLastQueriedHeight = height
}

// SetLatestClientState updates client state
func (r *Relayer) SetLatestClientState(clientState *celestia.ClientState) {
	r.latestClientState = clientState
}

// GetLocalBlockAtAHeight allows keeper package to use the relayer's provider to fetch its blocks
func (r *Relayer) GetLocalBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	return r.localProvider.GetBlockAtHeight(ctx, height)
}

// DecrementBlockProofCacheLimit will reduce the number of proofs injected into proposal by 1
func (r *Relayer) DecrementBlockProofCacheLimit() {
	r.blockProofCacheLimit = r.blockProofCacheLimit-1
}

// ClearUpdateClient will set update client to nil.
// This is done when the celestia light client has been updated.
// It is only populated when the trusting period is 2/3 time from expiration.
func (r *Relayer) ClearUpdateClient() {
	r.updateClient = nil
}