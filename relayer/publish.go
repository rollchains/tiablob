package relayer

import (
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/rollchains/tiablob/celestia/appconsts"
	blobtypes "github.com/rollchains/tiablob/celestia/blob/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/avast/retry-go/v4"
)

const (
	CelestiaPublishKeyName  = "blob"
	CelestiaFeegrantKeyName = "feegrant"
	celestiaBech32Prefix    = "celestia"
	celestiaBlobPostMemo    = "Posted by tiablob https://rollchains.com"
)

// PostNextBlocks is called by the current proposing validator during PrepareProposal.
// If on the publish boundary, it will return the block heights that will be published
// It will not publish the block being proposed.
func (r *Relayer) ProposePostNextBlocks(ctx sdk.Context, provenHeight int64) []int64 {
	height := ctx.BlockHeight()

	if height <= 1 {
		return nil
	}

	// only publish new blocks on interval
	if (height-1)%int64(r.celestiaPublishBlockInterval) != 0 {
		return nil
	}

	var blocks []int64
	for block := height - int64(r.celestiaPublishBlockInterval); block < height; block++ {
		// this could be false after a genesis restart
		if block > provenHeight {
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// PostBlocks is call in the preblocker, the proposer will publish at this point with their block accepted
func (r *Relayer) PostBlocks(ctx sdk.Context, blocks []int64) {
	go r.postBlocks(ctx, blocks)
}

// postBlocks will publish rollchain blocks to celestia
// start height is inclusive, end height is exclusive
func (r *Relayer) postBlocks(ctx sdk.Context, blocks []int64) {
	if len(blocks) == 0 {
		return
	}

	if !r.celestiaProvider.KeyExists(CelestiaPublishKeyName) {
		r.logger.Error("No Celestia key found, please add with `tiablob keys add` command")
		return
	}

	// if not set, will not use feegrant
	feeGranter, _ := r.celestiaProvider.GetKeyAddress(CelestiaFeegrantKeyName)

	blobs := make([]*blobtypes.Blob, len(blocks))

	for i, height := range blocks {
		var blockBz []byte
		res, err := r.localProvider.GetBlockAtHeight(ctx, height)
		if err != nil {
			r.logger.Error("Error getting block via rpc", "height", height, "error", err)
			// If block has been pruned, get it from unproven block store which isn't pruned until proven, but can lag behind cometbft's blockstore
			blockBz, err = r.GetLocalBlockAtHeight(height)
			if err != nil {
				r.logger.Error("Error getting block from unproven blockstore", "height", height, "error", err)
				return
			}
		} else {
			blockProto, err := res.Block.ToProto()
			if err != nil {
				r.logger.Error("Error protoing block", "error", err)
				return
			}

			blockBz, err = blockProto.Marshal()
			if err != nil {
				r.logger.Error("Error marshaling block", "error", err)
				return
			}
		}

		blob, err := blobtypes.NewBlob(r.celestiaNamespace, blockBz, appconsts.ShareVersionZero)
		if err != nil {
			r.logger.Error("Error building blob", "error", err)
			return
		}

		blobs[i] = blob
	}

	blobLens := make([]uint32, len(blocks))
	for i, blob := range blobs {
		blobLens[i] = uint32(len(blob.Data))
	}

	gasLimit := blobtypes.DefaultEstimateGas(blobLens)
	if feeGranter != nil {
		// 12000 required for feegrant
		gasLimit = gasLimit + 12000
	}

	signer, err := r.celestiaProvider.ShowAddress(CelestiaPublishKeyName, "celestia")
	if err != nil {
		r.logger.Error("Error getting signer address", "error", err)
		return
	}

	msg, err := blobtypes.NewMsgPayForBlobs(signer, blobs...)
	if err != nil {
		r.logger.Error("Error building MsgPayForBlobs", "error", err)
		return
	}

	ws := r.celestiaProvider.EnsureWalletState(CelestiaPublishKeyName)
	ws.Mu.Lock()
	defer ws.Mu.Unlock()

	if err := retry.Do(func() error {
		seq, txBz, err := r.celestiaProvider.Sign(
			ctx,
			ws,
			r.celestiaChainID,
			r.celestiaGasPrice,
			r.celestiaGasAdjustment,
			gasLimit,
			celestiaBech32Prefix,
			CelestiaPublishKeyName,
			feeGranter,
			[]sdk.Msg{msg},
			celestiaBlobPostMemo,
		)
		if err != nil {
			// Account sequence mismatch errors can happen on the simulated transaction also.
			if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
				ws.HandleAccountSequenceMismatchError(err)
			}
			return fmt.Errorf("Error signing blob tx, %w", err)
		}

		blobTx, err := tmtypes.MarshalBlobTx(txBz, blobs...)
		if err != nil {
			return fmt.Errorf("Error marshaling blob tx, %w", err)
		}

		res, err := r.celestiaProvider.Broadcast(ctx, seq, ws, blobTx)
		if err != nil {
			if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
				ws.HandleAccountSequenceMismatchError(err)
			}
			return fmt.Errorf("Error broadcasting blob tx, %w", err)
		}

		r.logger.Info("Posted block(s) to Celestia",
			"height_start", blocks[0],
			"height_end", blocks[len(blocks)-1],
			"celestia_height", res.Height,
			"namespace", string(r.celestiaNamespace.ID[18:]),
			"tx_hash", hex.EncodeToString(res.Hash),
			"url", fmt.Sprintf("https://mocha.celenium.io/tx/%s", hex.EncodeToString(res.Hash)),
		)

		return nil
	}, retry.Context(ctx), retry.Attempts(uint(2)), retry.OnRetry(func(n uint, err error) {
		r.logger.Info(
			"Failed to published blobs",
			"attempt", n+1,
			"error", err,
		)
	})); err != nil {
		r.logger.Error("Error blobs not published")
	}
}
