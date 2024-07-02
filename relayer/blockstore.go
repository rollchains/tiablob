package relayer

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
)

func (r *Relayer) GetLocalBlockAtHeight(height int64) ([]byte, error) {
	var block []byte
	if err := retry.Do(func() error { 
		block = r.unprovenBlockStore.LoadBlock(height)
		if block == nil {
			return fmt.Errorf("loading unproven block is nil at height: %d", height)
		}
		return nil
	}, retry.Delay(time.Millisecond * 50),retry.Attempts(uint(10)), retry.OnRetry(func(n uint, err error) {
		r.logger.Info("failed to get block from unproven store", "height", height, "attempt", n+1)
		if n == 2 || n == 6 {
			r.PopulateUnprovenBlockStore()
		}
	})); err != nil {
		return nil, err
	}

	return block, nil
}

// Called in Preblocker, gets the latest block(s) from cometbft and populates our store
func (r *Relayer) PopulateUnprovenBlockStore() {
	ctx := context.Background()

	if r.clientCtx.Client == nil {
		r.logger.Info("comet rpc client is not set yet")
		return
	}
	abciInfo, err := r.clientCtx.Client.ABCIInfo(ctx)
	if err != nil {
		r.logger.Info("unable to get latest height, not an error", "error", err)
		return
	}
	lastBlockHeight := abciInfo.Response.LastBlockHeight

	currentBlockHeight := r.unprovenBlockStore.Height()
	r.logger.Info("Executing populate unproven blockstore", "initial height", currentBlockHeight, "latest height", lastBlockHeight)
	for height := currentBlockHeight + 1; height <= lastBlockHeight; height++ {
		// Cannot use txClient.GetBlockWithTxs since it tries to decode the txs. This API is broken when using the same tx
		// injection method as vote extensions. https://docs.cosmos.network/v0.50/build/abci/vote-extensions#vote-extension-propagation
		// "FinalizeBlock will ignore any byte slice that doesn't implement an sdk.Tx, so any injected vote extensions will safely be ignored in FinalizeBlock"
		// "Some existing Cosmos SDK core APIs may need to be modified and thus broken."
		block, err := r.localProvider.GetBlockAtHeight(ctx, height)
		if err != nil {
			r.logger.Error("getting local block", "height", height)
			return
		}

		blockProto, err := block.Block.ToProto()
		if err != nil {
			r.logger.Error("block to proto for unproven block store", "error", err)
			return
		}

		blockBz, err := blockProto.Marshal()
		if err != nil {
			r.logger.Error("block proto marshal", "error", err)
		}

		r.unprovenBlockStore.SaveBlock(height, blockBz)
	}
	r.logger.Info("populate unproven blockstore done")

}

func (r *Relayer) PushSnapshotBlocks(height int64, block []byte) {
	r.unprovenBlockStore.SaveBlock(height, block)
}

// Only prune to the previous proven height when proven height changes
// on startup, the last block is replayed, those blocks need to persist for the replay
func (r *Relayer) PruneBlockStore(previousProvenHeight int64) {
	r.unprovenBlockStore.PruneBlocks(previousProvenHeight)
}