package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/lightclients/celestia"
)

const DelayAfterUpgrade = int64(10)

func (k Keeper) prepareInjectData(ctx sdk.Context, currentBlockTime time.Time, latestProvenHeight int64) InjectedData {
	return InjectedData{
		CreateClient:  k.prepareCreateClient(ctx),
		PendingBlocks: k.preparePostBlocks(ctx, currentBlockTime),
		Proofs:        k.relayer.GetCachedProofs(k.injectedProofsLimit, latestProvenHeight),
		Headers:       k.relayer.GetCachedHeaders(k.injectedProofsLimit, latestProvenHeight),
	}
}

func (k Keeper) addTiablobDataToTxs(injectDataBz []byte, maxTxBytes int64, txs [][]byte) [][]byte {
	if len(injectDataBz) > 0 {
		var finalTxs [][]byte
		totalTxBytes := int64(len(injectDataBz))
		fmt.Println("Injected data size: ", totalTxBytes) // TODO: remove, debug only
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
	// Call PostNextBlocks to publish next blocks (if necessary)
	newBlocks := k.relayer.ProposePostNextBlocks(ctx)

	// If there are no new blocks to propose, check for expired blocks
	// Additionally, if the block interval is 1, we need to also be able to re-publish an expired block
	if len(newBlocks) < 2 && k.shouldGetExpiredBlock(ctx) {
		expiredBlocks := k.GetExpiredBlocks(ctx, currentBlockTime)
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

// shouldGetExpiredBlocks checks if this chain has recently upgraded.
// If so, it will delay publishing expired blocks so that the relayer has time to populate block proof cache first
func (k Keeper) shouldGetExpiredBlock(ctx sdk.Context) bool {
	_, lastUpgradeHeight, _ := k.upgradeKeeper.GetLastCompletedUpgrade(ctx)
	return ctx.BlockHeight() >= lastUpgradeHeight+DelayAfterUpgrade
}
