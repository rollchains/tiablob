package relayer

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/go-square/blob"
	"github.com/celestiaorg/go-square/inclusion"
	"github.com/celestiaorg/go-square/namespace"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/rollchains/tiablob/relayer/cosmos"
)

// Relayer is responsible for posting new blocks to Celestia and relaying block proofs from Celestia via the current proposer
type Relayer struct {
	provenHeights      chan uint64
	latestProvenHeight uint64

	commitHeights      chan uint64
	latestCommitHeight uint64

	pollInterval time.Duration

	blockProofCache      map[uint64][]byte
	blockProofCacheLimit int

	provider *cosmos.CosmosProvider
}

func NewRelayer(
	celestiaRpcUrl string,
	keyDir string,
	latestProvenHeight uint64,
	latestCommitHeight uint64,
	pollInterval time.Duration,
	blockProofCacheLimit int,
) (*Relayer, error) {
	provider, err := cosmos.NewProvider(celestiaRpcUrl, keyDir)
	if err != nil {
		return nil, err
	}

	return &Relayer{
		pollInterval: pollInterval,

		provenHeights:      make(chan uint64, 10000),
		latestProvenHeight: uint64(latestProvenHeight),

		commitHeights:      make(chan uint64, 10000),
		latestCommitHeight: uint64(latestProvenHeight),

		provider: provider,

		blockProofCache:      make(map[uint64][]byte),
		blockProofCacheLimit: blockProofCacheLimit,
	}, nil
}

func (r *Relayer) Start(ctx context.Context) error {
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

func (r *Relayer) NotifyCommitHeight(height uint64) {
	r.provenHeights <- height
}

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

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context) {
	celestiaHeight, err := r.provider.QueryLatestHeight(ctx)
	if err != nil {
		//r.logger.Error("Error querying latest height from Celestia", "error", err)
		return
	}

	for i := r.latestProvenHeight + 1; i <= r.latestCommitHeight; i++ {

		blob := blob.New(namespace.PayForBlobNamespace, []byte(fmt.Sprintf("TODO populate with block data for height i: %d", i)), 0)

		// generate share commitment for height i
		commitment, err := inclusion.CreateCommitment(blob, merkle.HashFromByteSlices, 64)
		if err != nil {
			//r.logger.Error("error creating commitment for height", "height", i, "error", err)
			break
		}

		// TODO confirm all of these details. Looks like the share commitment may not be able to be queried in the typical way
		res, err := r.provider.QueryABCI(ctx, abci.RequestQuery{
			Path:   "store/blob/key",
			Data:   commitment,
			Height: celestiaHeight,
			Prove:  true,
		})
		if err != nil {
			//r.logger.Error("error querying share commitment proof for height", "height", i, "error", err)
			break
		}

		if res.Code != 0 {
			//r.logger.Error("error querying share commitment proof for height", "height", i, "code", res.Code, "log", res.Log)
			break
		}

		// cache the proof
		//r.blockProofCache[i] = res.ProofOps.Ops
	}
}

// Reconcile is intended to be called by the current proposing validator during PrepareProposal and will:
// - check the cache (no fetches here to minimize duration) if there are any new block proofs to be relayed from Celestia
// - if there are any block proofs to relay, generate a MsgUpdateClient along with the MsgProveBlock message(s) and return them in a tx for injection into the proposal.
// - if the Celestia light client is within 1/3 of the trusting period and there are no block proofs to relay, generate a MsgUpdateClient to update the light client and return it in a tx.
// - in a non-blocking goroutine, post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
func (r *Relayer) Reconcile(ctx context.Context) [][]byte {
	go r.postNextBlocks(ctx)

	panic("implement me")
}

// postNextBlocks will post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
// Skip the last n blocks to give time for the previous proposer's transaction to succeed.
func (r *Relayer) postNextBlocks(ctx context.Context) {
	panic("implement me")
}
