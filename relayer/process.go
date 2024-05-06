package relayer

import (
	"context"
	"fmt"
	"time"
)

// Start begins the relayer process
func (r *Relayer) Start(
	ctx context.Context,
	provenHeight uint64,
	commitHeight uint64,
) error {
	r.latestProvenHeight = provenHeight
	r.latestCommitHeight = commitHeight

	if err := r.celestiaProvider.CreateKeystore(); err != nil {
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
			r.shouldUpdateClient(ctx)
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