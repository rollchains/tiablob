package relayer

import (
	"context"
	"time"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (r *Relayer) getCachedHeader(height int64) *celestia.Header {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.celestiaHeaderCache[height]
}

func (r *Relayer) setCachedHeader(height int64, header *celestia.Header) {
	r.mu.Lock()
	r.celestiaHeaderCache[height] = header
	r.mu.Unlock()
}

func (r *Relayer) GetUpdateClient() *celestia.Header {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.updateClient
}

// A non-nil set is performed when trusting period is 2/3 time from expiration
// a nil clears it
func (r *Relayer) SetUpdateClient(updateClient *celestia.Header) {
	r.mu.Lock()
	r.updateClient = updateClient
	r.mu.Unlock()
}

// GetCachedHeaders queries through the cached block proofs, extracts the celestia height they were published, and
// pulls in the cached header. If a header does not exist, it may have already been included in an earlier block.
// Finally, if there are no header/client updates, we check if an update client is needed to keep the client from expiring.
func (r *Relayer) GetCachedHeaders(proofLimit int, latestProvenHeight int64) []*celestia.Header {
	headersMap := make(map[int64]*celestia.Header)
	numProofs := 0
	checkHeight := latestProvenHeight + 1
	proof := r.getCachedProof(checkHeight)
	for proof != nil {
		header := r.getCachedHeader(proof.CelestiaHeight)
		if header != nil {
			headersMap[proof.CelestiaHeight] = header
		}
		numProofs++
		if numProofs >= proofLimit {
			break
		}
		checkHeight = checkHeight + int64(len(proof.RollchainHeights))
		proof = r.getCachedProof(checkHeight)
	}

	var headers []*celestia.Header
	for _, header := range headersMap {
		headers = append(headers, header)
	}

	// If no headers, check for an update client (trusting period has <1/3 left)
	if len(headers) == 0 {
		updateClient := r.GetUpdateClient()
		if updateClient != nil {
			headers = append(headers, updateClient)
		}
	}

	return headers
}

// FetchNewHeader will generate a celestia light client header for updating the client.
// Headers will either be generated for a proof or to keep the client from expiring.
func (r *Relayer) fetchNewHeader(ctx context.Context, queryHeight int64, latestClientState *celestia.ClientState) *celestia.Header {
	newLightBlock, err := r.celestiaProvider.QueryLightBlock(ctx, queryHeight)
	if err != nil {
		r.logger.Error("error querying light block for proofs", "height", queryHeight)
		return nil
	}

	if latestClientState == nil {
		r.logger.Error("Client state is not set")
		return nil
	}
	trustedHeight := latestClientState.LatestHeight

	trustedValidatorsInBlock, err := r.celestiaProvider.QueryLightBlock(ctx, int64(trustedHeight.RevisionHeight+1))
	if err != nil {
		r.logger.Error("error querying trusted light block", "height", trustedHeight)
		return nil
	}

	valSet, err := newLightBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error new light block to val set proto")
		return nil
	}

	trustedValSet, err := trustedValidatorsInBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error trusted validators in block to val set proto")
		return nil
	}

	return &celestia.Header{
		SignedHeader:      newLightBlock.SignedHeader.ToProto(),
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedValSet,
	}
}

// shouldUpdateClient checks if we need to update the client with a new consensus state to avoid expiration
// if an update is needed, it will be cached in r.updateClient which will be cleared whenever the client is updated
func (r *Relayer) shouldUpdateClient(ctx context.Context, latestClientState *celestia.ClientState) {
	if latestClientState == nil {
		// No error, just no client state yet, nothing to update
		return
	}

	cHeader, err := r.celestiaProvider.QueryLightBlock(ctx, int64(latestClientState.LatestHeight.RevisionHeight))
	if err != nil {
		r.logger.Error("shouldUpdateClient: querying latest client state light block", "error", err)
		return
	}

	twoThirdsTrustingPeriodMs := float64(latestClientState.TrustingPeriod.Milliseconds()) * 2 / 3
	timeSinceLastClientUpdateMs := float64(time.Since(cHeader.Header.Time).Milliseconds())

	if timeSinceLastClientUpdateMs > twoThirdsTrustingPeriodMs {
		celestiaLatestHeight, err := r.celestiaProvider.QueryLatestHeight(ctx)
		if err != nil {
			r.logger.Error("shouldUpdateClient: querying latest height from Celestia", "error", err)
			return
		}
		updateClient := r.fetchNewHeader(ctx, celestiaLatestHeight, latestClientState)
		r.SetUpdateClient(updateClient)
	}
}
