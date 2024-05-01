package relayer

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	
	"github.com/rollchains/tiablob/lightclients/celestia"
)

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
	r.commitHeights <- height
}

// NotifyProvenHeight is called by the app to notify the relayer of the latest proven height
// i.e. the height of the highest incremental block that was proven to be posted to Celestia.
func (r *Relayer) NotifyProvenHeight(height uint64) {
	r.provenHeights <- height
}

// updateHeight is called when the provenHeight has changed
func (r *Relayer) updateHeight(height uint64) {
	if height > r.latestProvenHeight {
		fmt.Println("Latest proven height:", height)
		r.latestProvenHeight = height
		r.pruneCache(height)
	}
}

// pruneCache will delete any headers or proofs that are no longer needed
func (r *Relayer) pruneCache(provenHeight uint64) {
	for h, proof := range r.blockProofCache {
		if h <= provenHeight {
			if r.celestiaHeaderCache[proof.CelestiaHeight] != nil {
				delete(r.celestiaHeaderCache, proof.CelestiaHeight)
			}
			// this height has been proven (either by us or another proposer), we can delete the cached proof
			delete(r.blockProofCache, h)
		}
	}
}

// Reconcile is intended to be called by the current proposing validator during PrepareProposal and will:
//   - call a non-blocking gorouting to post the next block (or chunk of blocks) above the last proven height to Celestia (does
//     not include the block being proposed).
//   - check the proofs cache (no fetches here to minimize duration) if there are any new block proofs to be relayed from Celestia
//   - if there are any block proofs to relay, it will add any headers (update clients) that are also cached
//   - if the Celestia light client is within 1/3 of the trusting period and there are no block proofs to relay, generate a
//     MsgUpdateClient to update the light client and return it in a tx.
func (r *Relayer) Reconcile(ctx sdk.Context) ([]*celestia.BlobProof, []*celestia.Header) {
	go r.postNextBlocks(ctx, r.celestiaPublishBlockInterval)

	proofs := r.GetCachedProofs()
	headers := r.GetCachedHeaders()

	// TODO: check if the light client is within 1/3 of the trusting period, if so, add header
	if len(headers) == 0 {
		// check trust period and add if necessary
	}

	return proofs, headers
}
