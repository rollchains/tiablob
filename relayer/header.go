package relayer

import (
	"context"

	"github.com/rollchains/tiablob/light-clients/celestia"
)

func (r *Relayer) GetCachedHeaders() []*Header{
	clientsMap := make(map[uint64]*Header)

	for checkHeight := r.latestProvenHeight + 1; r.blockProofCache[checkHeight] != nil; checkHeight++ {
		proof := r.blockProofCache[checkHeight]
		if r.celestiaHeaderCache[proof.CelestiaHeight] != nil {
			clientsMap[proof.CelestiaHeight] = r.celestiaHeaderCache[proof.CelestiaHeight]
		}
	}
	
	clients := []*Header{}
	for _, header := range clientsMap {
		clients = append(clients, header)
	}

	return clients
}

func (r *Relayer) FetchNewHeader(ctx context.Context, queryHeight uint64) *Header {
	newLightBlock, err := r.provider.QueryLightBlock(ctx, int64(queryHeight))
	if err != nil {
		r.logger.Error("error querying light block for proofs, height:", queryHeight)
		return nil
	}

	var trustedHeight celestia.Height
	if r.latestClientState != nil {
		trustedHeight = r.latestClientState.LatestHeight
	} else {
		trustedHeight = celestia.NewHeight(1, 1)
	}
	if r.latestClientState == nil {
		r.logger.Error("Client state is not set")
		return nil
	}

	trustedValidatorsInBlock, err := r.provider.QueryLightBlock(ctx, int64(trustedHeight.GetRevisionHeight()+1))
	if err != nil {
		r.logger.Error("error querying trusted light block, height:", trustedHeight)
		return nil
	}

	valSet, err := newLightBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error new light block to val set proto")
		return nil
	}

	valSetBz, err := valSet.Marshal()
	if err != nil {
		r.logger.Error("error val set marshal")
		return nil
	}

	trustedValSet, err := trustedValidatorsInBlock.ValidatorSet.ToProto()
	if err != nil {
		r.logger.Error("error trusted validators in block to val set proto")
		return nil
	}

	trustedValSetBz, err := trustedValSet.Marshal()
	if err != nil {
		r.logger.Error("error trusted val set marshal")
		return nil
	}

	return &Header{
		SignedHeader: newLightBlock.SignedHeader.ToProto(),
		ValidatorSet: valSetBz,
		TrustedHeight: trustedHeight,
		TrustedValidators: trustedValSetBz,
	}
}