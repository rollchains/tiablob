package relayer

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/celestia-node/blob"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/light-clients/celestia"
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

	blockProofCache      map[uint64]*Proof
	blockProofCacheLimit int
	consensusStateCache  map[uint64]*celestia.Header

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
		celestiaLastQueriedHeight:    0, // Start at 0 for now, but we'll get this from the latest client state

		nodeRpcUrl:    cfg.NodeRpcURL,
		nodeAuthToken: cfg.NodeAuthToken,

		blockProofCache:      make(map[uint64]*Proof),
		blockProofCacheLimit: cfg.MaxFlushSize,
		consensusStateCache:  make(map[uint64]*celestia.Header),
	}, nil
}

func (r *Relayer) SetClientContext(clientCtx client.Context) {
	r.clientCtx = clientCtx
}

func (r *Relayer) SetLatestClientState(clientState *celestia.ClientState) {
	r.latestClientState = clientState
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

type (
	InjectClientData struct {
		CreateClient *CreateClient
		UpdateClientAndProofs []*UpdateClientAndProofs
	}
	CreateClient struct {
		ClientState celestia.ClientState
		ConsensusState celestia.ConsensusState
	}
	Proof struct {
		Proof *blob.Proof
		Height uint64 // Consensus state height
		Commitment *blob.Commitment
	}
	UpdateClientAndProofs struct {
		UpdateClient *celestia.ClientMessage
		Proofs []*Proof
	}
)

// Reconcile is intended to be called by the current proposing validator during PrepareProposal and will:
// - check the cache (no fetches here to minimize duration) if there are any new block proofs to be relayed from Celestia
// - if there are any block proofs to relay, generate a MsgUpdateClient along with the MsgProveBlock message(s) and return them in a tx for injection into the proposal.
// - if the Celestia light client is within 1/3 of the trusting period and there are no block proofs to relay, generate a MsgUpdateClient to update the light client and return it in a tx.
// - in a non-blocking goroutine, post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
func (r *Relayer) Reconcile(ctx sdk.Context, clientFound bool) InjectClientData {
	go r.postNextBlocks(ctx, r.celestiaPublishBlockInterval)

	var injectClientData InjectClientData
	if !clientFound {
		fmt.Println("Client not found, creating...")
		injectClientData.CreateClient = r.CreateClient(ctx)
	}

	// TODO check cache for new block proofs to relay
	return injectClientData
}
