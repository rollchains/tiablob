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
func (r *Relayer) ProposePostNextBlocks(ctx sdk.Context) []int64 {
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
		blocks = append(blocks, block)
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
		res, err := r.localProvider.GetBlockAtHeight(ctx, height)
		if err != nil {
			r.logger.Error("Error getting block", "height:", height, "error", err)
			return
		}

		blockProto, err := res.Block.ToProto()
		if err != nil {
			r.logger.Error("Error protoing block", "error", err)
			return
		}

		blockBz, err := blockProto.Marshal()
		if err != nil {
			r.logger.Error("Error marshaling block", "error", err)
			return
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

		r.logger.Error("Error signing blob tx", "error", err)
		return
	}

	blobTx, err := tmtypes.MarshalBlobTx(txBz, blobs...)
	if err != nil {
		r.logger.Error("Error marshaling blob tx", "error", err)
		return
	}

	res, err := r.celestiaProvider.Broadcast(ctx, seq, ws, blobTx)
	if err != nil {
		if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
			ws.HandleAccountSequenceMismatchError(err)
		}

		r.logger.Error("Error broadcasting blob tx", "error", err)

		return
	}

	r.logger.Info("Posted block(s) to Celestia",
		"height_start", blocks[0],
		"height_end", blocks[len(blocks)-1],
		"celestia_height", res.Height,
		"tx_hash", hex.EncodeToString(res.Hash),
		"url", fmt.Sprintf("https://mocha.celenium.io/tx/%s", hex.EncodeToString(res.Hash)),
	)
}
