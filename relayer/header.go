package relayer

import (
	"context"
	"time"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

// GetCachedHeaders queries through the cached block proofs, extracts the celestia height they were published, and
// pulls in the cached header. If a header does not exist, it may have already been included in an earlier block.
// Finally, if there are no header/client updates, we check if an update client is needed to keep the client from expiring.
func (r *Relayer) GetCachedHeaders() []*celestia.Header {
	headersMap := make(map[uint64]*celestia.Header)
	numProofs := 0
	for checkHeight := r.latestProvenHeight + 1; r.blockProofCache[checkHeight] != nil; checkHeight++ {
		proof := r.blockProofCache[checkHeight]
		if r.celestiaHeaderCache[proof.CelestiaHeight] != nil {
			headersMap[proof.CelestiaHeight] = r.celestiaHeaderCache[proof.CelestiaHeight]
		}
		numProofs++
		if numProofs >= r.blockProofCacheLimit {
			break
		}
	}

	var headers []*celestia.Header
	for _, header := range headersMap {
		headers = append(headers, header)
	}

	// If no headers, check for an update client (trusting period has <1/3 left)
	if len(headers) == 0 {
		if r.updateClient != nil {
			headers = append(headers, r.updateClient)
		}
	}

	return headers
}

// FetchNewHeader will generate a celestia light client header for updating the client.
// Headers will either be generated for a proof or to keep the client from expiring.
func (r *Relayer) FetchNewHeader(ctx context.Context, queryHeight uint64) *celestia.Header {
	newLightBlock, err := r.celestiaProvider.QueryLightBlock(ctx, int64(queryHeight))
	if err != nil {
		r.logger.Error("error querying light block for proofs", "height", queryHeight)
		return nil
	}

	if r.latestClientState == nil {
		r.logger.Error("Client state is not set")
		return nil
	}
	trustedHeight := r.latestClientState.LatestHeight

	trustedValidatorsInBlock, err := r.celestiaProvider.QueryLightBlock(ctx, int64(trustedHeight.GetRevisionHeight()+1))
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

// shouldUpdateClient check if we need to update the client with a new consensus state to avoid expiration
// if an update is needed, it will get cached in r.updateClient which will be cleared whenever the client is updated
func (r *Relayer) shouldUpdateClient(ctx context.Context) {
	if r.latestClientState == nil {
		// No error, just no client state yet, nothing to update
		return
	}
	
	height := r.latestClientState.LatestHeight

	cHeader, err := r.celestiaProvider.QueryLightBlock(ctx, int64(height.GetRevisionHeight()))
	if err != nil {
		r.logger.Error("shouldUpdateClient: querying latest client state light block", "error", err)
		return
	}

	twoThirdsTrustingPeriodMs := float64(r.latestClientState.TrustingPeriod.Milliseconds()) * 2 / 3
	timeSinceLastClientUpdateMs := float64(time.Since(cHeader.Header.Time).Milliseconds())

	if timeSinceLastClientUpdateMs > twoThirdsTrustingPeriodMs {
		celestiaLatestHeight, err := r.celestiaProvider.QueryLatestHeight(ctx)
		if err != nil {
			r.logger.Error("shouldUpdateClient: querying latest height from Celestia", "error", err)
			return
		}
		r.updateClient = r.FetchNewHeader(ctx, uint64(celestiaLatestHeight))
	}
}