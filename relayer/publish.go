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

// postNextBlocks will post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
// Skip the last n blocks to give time for the previous proposer's transaction to succeed.
func (r *Relayer) postNextBlocks(ctx sdk.Context, n int) {
	// TODO post blocks since last proven height
	height := ctx.BlockHeight()

	if height <= 1 {
		return
	}

	// only publish every n blocks
	if (height-1)%int64(n) != 0 {
		return
	}

	if !r.provider.KeyExists(CelestiaPublishKeyName) {
		r.logger.Error("No Celestia key found, please add with `tiablob keys add` command")
		return
	}

	// if not set, will not use feegrant
	feeGranter, _ := r.provider.GetKeyAddress(CelestiaFeegrantKeyName)

	blobs := make([]*blobtypes.Blob, n)

	for i := 0; i < n; i++ {
		h := height - int64(n) + int64(i)
		res, err := r.provider.GetLocalBlockAtHeight(ctx, h)
		if err != nil {
			r.logger.Error("Error getting block", "height:", h, "error", err)
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

	blobLens := make([]uint32, n)
	for i, blob := range blobs {
		blobLens[i] = uint32(len(blob.Data))
	}

	gasLimit := blobtypes.DefaultEstimateGas(blobLens)
	if feeGranter != nil {
		// 12000 required for feegrant
		gasLimit = gasLimit + 12000
	}

	signer, err := r.provider.ShowAddress(CelestiaPublishKeyName, "celestia")
	if err != nil {
		r.logger.Error("Error getting signer address", "error", err)
		return
	}

	msg, err := blobtypes.NewMsgPayForBlobs(signer, blobs...)
	if err != nil {
		r.logger.Error("Error building MsgPayForBlobs", "error", err)
		return
	}

	ws := r.provider.EnsureWalletState(CelestiaPublishKeyName)
	ws.Mu.Lock()
	defer ws.Mu.Unlock()

	seq, txBz, err := r.provider.Sign(
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

	res, err := r.provider.Broadcast(ctx, seq, ws, blobTx)
	if err != nil {
		if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
			ws.HandleAccountSequenceMismatchError(err)
		}

		r.logger.Error("Error broadcasting blob tx", "error", err)

		return
	}

	r.logger.Info("Posted block(s) to Celestia",
		"height_start", height-int64(n),
		"height_end", height-1,
		"celestia_height", res.Height,
		"tx_hash", hex.EncodeToString(res.Hash),
		"url", fmt.Sprintf("https://mocha.celenium.io/tx/%s", hex.EncodeToString(res.Hash)),
	)
}
