package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/lightclients/celestia"

	"github.com/rollchains/tiablob"
)

const DelayAfterUpgrade = int64(10)

func (k *Keeper) prepareInjectData(ctx sdk.Context, currentBlockTime time.Time, latestProvenHeight int64, proposerAddr []byte) tiablob.MsgInjectedData {
	return tiablob.MsgInjectedData{
		CreateClient:  k.prepareCreateClient(ctx),
		PendingBlocks: k.preparePostBlocks(ctx, currentBlockTime),
		Proofs:        k.relayer.GetCachedProofs(k.injectedProofsLimit, latestProvenHeight),
		Headers:       k.relayer.GetCachedHeaders(k.injectedProofsLimit, latestProvenHeight),
		ProposerAddress: proposerAddr,
		BlockTime: currentBlockTime,
		Signer: "rc10d07y265gmmuvt4z0w9aw880jnsr700jymjvfq",
	}
}

func (k *Keeper) addTiablobDataToTxs(injectDataBz []byte, maxTxBytes int64, txs [][]byte) [][]byte {
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

func (k *Keeper) prepareCreateClient(ctx sdk.Context) *celestia.CreateClient {
	// If there is no client state, create a client
	_, clientFound := k.GetClientState(ctx)
	if !clientFound {
		return k.relayer.CreateClient(ctx)
	}
	return nil
}

func (k *Keeper) preparePostBlocks(ctx sdk.Context, currentBlockTime time.Time) tiablob.PendingBlocks {
	provenHeight, err := k.GetProvenHeight(ctx)
	if err != nil {
		return tiablob.PendingBlocks{}
	}
	newBlocks := k.relayer.ProposePostNextBlocks(ctx, provenHeight)

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

	return tiablob.PendingBlocks{
		BlockHeights: newBlocks,
	}
}

// shouldGetExpiredBlocks checks if this chain has recently upgraded.
// If so, it will delay publishing expired blocks so that the relayer has time to populate block proof cache first
func (k *Keeper) shouldGetExpiredBlock(ctx sdk.Context) bool {
	_, lastUpgradeHeight, _ := k.upgradeKeeper.GetLastCompletedUpgrade(ctx)
	return ctx.BlockHeight() >= lastUpgradeHeight+DelayAfterUpgrade
}

// marshalMaxBytes will marshal the injected data ensuring it fits within the max bytes.
// If it does not fit, it will decrement the number of proofs to inject by 1 and retry.
// This new configuration will persist until the node is restarted. If a decrement is required,
// there was most likely a misconfiguration for block proof cache limit.
// Injected data is roughly 1KB/proof
func (k *Keeper) marshalMaxBytes(injectData *tiablob.MsgInjectedData, maxBytes int64, latestProvenHeight int64) []byte {
	if injectData.IsEmpty() {
		return nil
	}

	tx := tiablob.NewInjectTx(k.cdc, []sdk.Msg{injectData})

	injectDataBz, err := k.cdc.Marshal(tx.Tx)
	if err != nil {
		return nil
	}

	proofLimit := k.injectedProofsLimit
	for int64(len(injectDataBz)) > maxBytes {
		proofLimit = proofLimit - 1
		injectData.Proofs = k.relayer.GetCachedProofs(proofLimit, latestProvenHeight)
		injectData.Headers = k.relayer.GetCachedHeaders(proofLimit, latestProvenHeight)
		if len(injectData.Proofs) == 0 {
			return nil
		}
		tx := tiablob.NewInjectTx(k.cdc, []sdk.Msg{injectData})

		injectDataBz, err = k.cdc.Marshal(tx.Tx)
		if err != nil {
			return nil
		}
	}

	return injectDataBz
}
