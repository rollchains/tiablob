package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (k Keeper) proposeCreateClient(ctx sdk.Context) *celestia.CreateClient {
	// If there is no client state, create a client
	_, clientFound := k.GetClientState(ctx)
	if !clientFound {
		return k.relayer.CreateClient(ctx)
	}
	return nil
}

func (k Keeper) proposePostBlocks(ctx sdk.Context, currentBlockTime time.Time) PendingBlocks {
	// Call PostNextBlocks to publish next blocks (if necessary) and/or retry timed-out published blocks
	newBlocks := k.relayer.ProposePostNextBlocks(ctx)
	
	// If there are no new blocks to propose, check for expired blocks
	// Additionally, if the block interval is 1, we need to also be able to re-publish an expired block
	if len(newBlocks) < 2 {
		expiredBlocks := k.GetExpiredBlocks(ctx, k.cdc, currentBlockTime)
		for _, expiredBlock := range expiredBlocks {
			// Check if we have a proof for this block already
			if k.relayer.HasCachedProof(expiredBlock) {
				continue
			}
			// Add it to the list respecting the publish limit
			if len(newBlocks) < 2 || len(newBlocks) < k.relayer.GetPublishBlockInterval() {
				newBlocks = append(newBlocks, expiredBlock)
			}
		}
	}

	return PendingBlocks{
		BlockHeight: newBlocks,
	}
}