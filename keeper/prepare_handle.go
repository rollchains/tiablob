package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

func (k Keeper) prepareInjectData(ctx sdk.Context, currentBlockTime time.Time) InjectedData {
	return InjectedData{
		CreateClient:  k.prepareCreateClient(ctx),
		PendingBlocks: k.preparePostBlocks(ctx, currentBlockTime),
		Proofs:        k.relayer.GetCachedProofs(),
		Headers:       k.relayer.GetCachedHeaders(),
	}
}

func (k Keeper) addTiablobDataToTxs(injectDataBz []byte, maxTxBytes int64, txs [][]byte) [][]byte {
	if len(injectDataBz) > 0 {
		var finalTxs [][]byte
		totalTxBytes := int64(len(injectDataBz))
		finalTxs = append(finalTxs, injectDataBz)
		for _, tx := range txs {
			totalTxBytes += int64(len(tx))
			// Append as many transactions as will fit
			if totalTxBytes <= maxTxBytes {
				finalTxs = append(finalTxs, tx)
			} else {
				break
			}
		}
		return finalTxs
	}
	return txs
}

func (k Keeper) prepareCreateClient(ctx sdk.Context) *celestia.CreateClient {
	// If there is no client state, create a client
	_, clientFound := k.GetClientState(ctx)
	if !clientFound {
		return k.relayer.CreateClient(ctx)
	}
	return nil
}

func (k Keeper) preparePostBlocks(ctx sdk.Context, currentBlockTime time.Time) PendingBlocks {
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
			if len(newBlocks) < 2 || len(newBlocks) < k.publishToCelestiaBlockInterval {
				newBlocks = append(newBlocks, expiredBlock)
			}
		}
	}

	return PendingBlocks{
		BlockHeights: newBlocks,
	}
}
