package relayer

import (
	"context"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/lightclients/celestia"
	"github.com/rollchains/tiablob/relayer/cosmos"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
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

	provider  *cosmos.CosmosProvider
	clientCtx client.Context

	latestClientState *celestia.ClientState

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

	provider, err := cosmos.NewProvider(cfg.AppRpcURL, keyDir, cfg.AppRpcTimeout, cfg.ChainID)
	if err != nil {
		return nil, err
	}

	return &Relayer{
		logger: logger,

		pollInterval: cfg.ProofQueryInterval,

		provenHeights: make(chan uint64, 10000),
		commitHeights: make(chan uint64, 10000),

		provider:                     provider,
		celestiaNamespace:            celestiaNamespace,
		celestiaChainID:              cfg.ChainID,
		celestiaGasPrice:             cfg.GasPrice,
		celestiaGasAdjustment:        cfg.GasAdjustment,
		celestiaPublishBlockInterval: celestiaPublishBlockInterval,
		celestiaLastQueriedHeight:    0, // TODO: Start at 0 for now, but we'll get this from the latest client state

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

// SetLatestClientState updates client state
func (r *Relayer) SetLatestClientState(clientState *celestia.ClientState) {
	r.latestClientState = clientState
}

// GetLocalBlockAtAHeight allows keeper package to use the relayer's provider to fetch its blocks
func (r *Relayer) GetLocalBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	return r.provider.GetLocalBlockAtHeight(ctx, height)
}

// DecrementBlockProofCacheLimit will reduce the number of proofs injected into proposal by 1
func (r *Relayer) DecrementBlockProofCacheLimit() {
	r.blockProofCacheLimit = r.blockProofCacheLimit-1
}