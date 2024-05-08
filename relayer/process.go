package relayer

import (
	"context"
	"fmt"
	"time"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

// Start begins the relayer process
func (r *Relayer) Start(
	ctx context.Context,
	provenHeight int64,
	commitHeight int64,
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
		case height := <-r.commitHeights:
			r.latestCommitHeight = height
		case height := <-r.provenHeights:
			r.updateHeight(height)
		case <-timer.C:
			if r.isValidatorAndCaughtUp(ctx) {
				latestClientState := r.getClientState(ctx)
				r.checkForNewBlockProofs(ctx, latestClientState)
				r.shouldUpdateClient(ctx, latestClientState)
			}
			timer.Reset(r.pollInterval)
		}
	}
}

// NotifyCommitHeight is called by the app to notify the relayer of the latest commit height
func (r *Relayer) NotifyCommitHeight(height int64) {
	r.commitHeights <- height
}

// NotifyProvenHeight is called by the app to notify the relayer of the latest proven height
// i.e. the height of the highest incremental block that was proven to be posted to Celestia.
func (r *Relayer) NotifyProvenHeight(height int64) {
	r.provenHeights <- height
}

// updateHeight is called when the provenHeight has changed
func (r *Relayer) updateHeight(height int64) {
	if height > r.latestProvenHeight {
		fmt.Println("Latest proven height:", height)
		r.latestProvenHeight = height
		r.pruneCache(height)
	}
}

// pruneCache will delete any headers or proofs that are no longer needed
func (r *Relayer) pruneCache(provenHeight int64) {
	r.mu.Lock()
	for h, proof := range r.blockProofCache {
		if h <= provenHeight {
			if r.celestiaHeaderCache[proof.CelestiaHeight] != nil {
				delete(r.celestiaHeaderCache, proof.CelestiaHeight)
			}
			// this height has been proven (either by us or another proposer), we can delete the cached proof
			delete(r.blockProofCache, h)
		}
	}
	r.mu.Unlock()
}

// only validators need to check for new block proofs and update clients
// if a validator, they should not be querying celestia until they finished catching up
func (r *Relayer) isValidatorAndCaughtUp(ctx context.Context) bool {
	status, err := r.localProvider.Status(ctx)
	if err != nil {
		return false
	}
	if status.ValidatorInfo.VotingPower > 0 && !status.SyncInfo.CatchingUp {
		return true
	}
	return false
}

func (r *Relayer) getClientState(ctx context.Context) *celestia.ClientState {
	// TODO: decide whether to query for the latest commit height, use r.latestCommitHeight, or use r.latestCommitHeight-1
	clientState, err := r.localProvider.QueryCelestiaClientState(ctx, r.latestCommitHeight-1)
	if err != nil {
		return nil
	}

	if clientState.LatestHeight.RevisionHeight == 0 {
		return nil
	}

	// update celestia's last queried height to avoid unnecessary queries
	if r.celestiaLastQueriedHeight < int64(clientState.LatestHeight.RevisionHeight) {
		r.celestiaLastQueriedHeight = int64(clientState.LatestHeight.RevisionHeight)
	}

	return clientState
}