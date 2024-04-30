package relayer

import (
	"context"

	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (r *Relayer) GetCachedHeaders() []*celestia.Header {
	clientsMap := make(map[uint64]*celestia.Header)
	for checkHeight := r.latestProvenHeight + 1; r.blockProofCache[checkHeight] != nil; checkHeight++ {
		proof := r.blockProofCache[checkHeight]
		if r.celestiaHeaderCache[proof.CelestiaHeight] != nil {
			clientsMap[proof.CelestiaHeight] = r.celestiaHeaderCache[proof.CelestiaHeight]
		}
	}

	var clients []*celestia.Header
	for _, header := range clientsMap {
		clients = append(clients, header)
	}

	return clients
}

func (r *Relayer) FetchNewHeader(ctx context.Context, queryHeight uint64) *celestia.Header {
	newLightBlock, err := r.provider.QueryLightBlock(ctx, int64(queryHeight))
	if err != nil {
		r.logger.Error("error querying light block for proofs", "height", queryHeight)
		return nil
	}

	if r.latestClientState == nil {
		r.logger.Error("Client state is not set")
		return nil
	}
	trustedHeight := r.latestClientState.LatestHeight

	trustedValidatorsInBlock, err := r.provider.QueryLightBlock(ctx, int64(trustedHeight.GetRevisionHeight()+1))
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
