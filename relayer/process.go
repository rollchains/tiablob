package relayer

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/relayer/cosmos"
)

// Relayer is responsible for posting new blocks to Celestia and relaying block proofs from Celestia via the current proposer
type Relayer struct {
	logger log.Logger

	provenHeights      chan uint64
	latestProvenHeight uint64

	commitHeights      chan uint64
	latestCommitHeight uint64

	pollInterval time.Duration

	blockProofCache      map[uint64][]byte
	blockProofCacheLimit int

	provider  *cosmos.CosmosProvider
	clientCtx client.Context

	celestiaChainID              string
	celestiaNamespace            appns.Namespace
	celestiaGasPrice             string
	celestiaGasAdjustment        float64
	celestiaPublishBlockInterval int
}

// NewRelayer creates a new Relayer instance
func NewRelayer(
	logger log.Logger,
	celestiaRpcUrl string,
	celestiaRpcTimeout time.Duration,
	celestiaNamespace appns.Namespace,
	celestiaGasPrice string,
	celestiaGasAdjustment float64,
	celestiaPublishBlockInterval int,
	keyDir string,
	pollInterval time.Duration,
	blockProofCacheLimit int,
) (*Relayer, error) {
	provider, err := cosmos.NewProvider(celestiaRpcUrl, keyDir, celestiaRpcTimeout)
	if err != nil {
		return nil, err
	}

	return &Relayer{
		logger: logger,

		pollInterval: pollInterval,

		provenHeights: make(chan uint64, 10000),
		commitHeights: make(chan uint64, 10000),

		provider:                     provider,
		celestiaNamespace:            celestiaNamespace,
		celestiaGasPrice:             celestiaGasPrice,
		celestiaGasAdjustment:        celestiaGasAdjustment,
		celestiaPublishBlockInterval: celestiaPublishBlockInterval,

		blockProofCache:      make(map[uint64][]byte),
		blockProofCacheLimit: blockProofCacheLimit,
	}, nil
}

func (r *Relayer) SetClientContext(clientCtx client.Context) {
	r.clientCtx = clientCtx
}

// Start begins the relayer process
func (r *Relayer) Start(
	ctx context.Context,
	provenHeight uint64,
	commitHeight uint64,
) error {
	r.latestProvenHeight = provenHeight
	r.latestCommitHeight = commitHeight

	if err := r.provider.CreateKeystore(); err != nil {
		return err
	}

	chainID, err := r.provider.QueryChainID(ctx)
	if err != nil {
		return fmt.Errorf("error querying celestia chain ID: %w", err)
	}

	r.celestiaChainID = chainID

	timer := time.NewTimer(r.pollInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case height := <-r.provenHeights:
			r.updateHeight(height)
		case <-timer.C:
			r.checkForNewBlockProofs(ctx)
			timer.Reset(r.pollInterval)
		}
	}
}

// NotifyCommitHeight is called by the app to notify the relayer of the latest commit height
func (r *Relayer) NotifyCommitHeight(height uint64) {
	r.provenHeights <- height
}

// NotifyProvenHeight is called by the app to notify the relayer of the latest proven height
// i.e. the height of the highest incremental block that was proven to be posted to Celestia.
func (r *Relayer) NotifyProvenHeight(height uint64) {
	r.provenHeights <- height
}

func (r *Relayer) updateHeight(height uint64) {
	if height > r.latestProvenHeight {
		r.latestProvenHeight = height

		for h := range r.blockProofCache {
			if h <= height {
				// this height has been proven (either by us or another proposer), we can delete the cached proof
				delete(r.blockProofCache, h)
			}
		}
	}
}

// Reconcile is intended to be called by the current proposing validator during PrepareProposal and will:
// - check the cache (no fetches here to minimize duration) if there are any new block proofs to be relayed from Celestia
// - if there are any block proofs to relay, generate a MsgUpdateClient along with the MsgProveBlock message(s) and return them in a tx for injection into the proposal.
// - if the Celestia light client is within 1/3 of the trusting period and there are no block proofs to relay, generate a MsgUpdateClient to update the light client and return it in a tx.
// - in a non-blocking goroutine, post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
func (r *Relayer) Reconcile(ctx sdk.Context) [][]byte {
	go r.postNextBlocks(ctx, r.celestiaPublishBlockInterval)

	// TODO check cache for new block proofs to relay
	return nil
}
