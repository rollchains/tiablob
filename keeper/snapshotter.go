package keeper

import (
	"context"
	"fmt"
	"io"

	errorsmod "cosmossdk.io/errors"
	snapshot "cosmossdk.io/store/snapshots/types"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/rollchains/tiablob"
)

var _ snapshot.ExtensionSnapshotter = &TiaBlobSnapshotter{}

// SnapshotFormat defines the default snapshot extension encoding format.
// SnapshotFormat 1 is a proto marshalled UnprovenBlock type.
const SnapshotFormat = 1

// TiaBlobSnapshotter implements the snapshot.ExtensionSnapshotter interface and is used to
// import and export unproven blocks so they can be proven when needed.
// State sync would otherwise missed these blocks and the node would panic.
type TiaBlobSnapshotter struct {
	cms    storetypes.MultiStore
	keeper *Keeper
}

// NewTiablobSnapshotter creates and returns a new snapshot.ExtensionSnapshotter implementation for tiablob.
func NewTiablobSnapshotter(cms storetypes.MultiStore, keeper *Keeper) snapshot.ExtensionSnapshotter {
	return &TiaBlobSnapshotter{
		cms:    cms,
		keeper: keeper,
	}
}

// SnapshotName implements the snapshot.ExtensionSnapshotter interface.
// A unique name should be provided such that the implementation can be identified by the manager.
func (*TiaBlobSnapshotter) SnapshotName() string {
	return tiablob.ModuleName
}

// SnapshotFormat implements the snapshot.ExtensionSnapshotter interface.
// This is the default format used for encoding payloads when taking a snapshot.
func (*TiaBlobSnapshotter) SnapshotFormat() uint32 {
	return SnapshotFormat
}

// SupportedFormats implements the snapshot.ExtensionSnapshotter interface.
// This defines a list of supported formats the snapshotter extension can restore from.
func (*TiaBlobSnapshotter) SupportedFormats() []uint32 {
	return []uint32{SnapshotFormat}
}

// SnapshotExtension implements the snapshot.ExntensionSnapshotter interface.
// SnapshotExtension is used to write data payloads into the underlying protobuf stream from the local client.
func (s *TiaBlobSnapshotter) SnapshotExtension(height uint64, payloadWriter snapshot.ExtensionPayloadWriter) error {
	ctx := context.Background()

	cacheMS, err := s.cms.CacheMultiStoreWithVersion(int64(height))
	if err != nil {
		return err
	}

	sdkCtx := sdk.NewContext(cacheMS, tmproto.Header{}, false, nil)

	provenHeight, err := s.keeper.GetProvenHeight(sdkCtx)
	if err != nil {
		return err
	}

	for unprovenHeight := provenHeight + 1; unprovenHeight <= int64(height); unprovenHeight++ {
		signedBlockBz, err := s.keeper.relayer.GetSignedBlockAtHeight(ctx, unprovenHeight)
		if err != nil {
			signedBlockBz = s.keeper.unprovenBlocks[unprovenHeight]
			if signedBlockBz == nil {
				return err
			}
		}

		unprovenBlock := tiablob.UnprovenBlock{
			Height: unprovenHeight,
			Block:  signedBlockBz,
		}

		unprovenBlockBz, err := unprovenBlock.Marshal()
		if err != nil {
			return err
		}

		if err = payloadWriter(unprovenBlockBz); err != nil {
			return err
		}
	}

	return nil
}

// RestoreExtension implements the snapshot.ExtensionSnapshotter interface.
// RestoreExtension is used to read data from an existing extension state snapshot into the tiablob keeper.
// The payload reader returns io.EOF when it has reached the end of the extension state snapshot.
func (s *TiaBlobSnapshotter) RestoreExtension(height uint64, format uint32, payloadReader snapshot.ExtensionPayloadReader) error {
	if format == s.SnapshotFormat() {
		return s.processAllItems(int64(height), payloadReader, restoreV1)
	}

	return errorsmod.Wrapf(snapshot.ErrUnknownFormat, "expected %d, got %d", s.SnapshotFormat(), format)
}

func (s *TiaBlobSnapshotter) processAllItems(
	height int64,
	payloadReader snapshot.ExtensionPayloadReader,
	cb func(*Keeper, int64, []byte) error,
) error {
	for {
		payload, err := payloadReader()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := cb(s.keeper, height, payload); err != nil {
			return errorsmod.Wrap(err, "failure processing snapshot item")
		}
	}

	return nil
}

func restoreV1(k *Keeper, height int64, unprovenBlockBz []byte) error {
	var unprovenBlock tiablob.UnprovenBlock
	if err := k.cdc.Unmarshal(unprovenBlockBz, &unprovenBlock); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal unproven block")
	}

	if unprovenBlock.Height > height {
		return fmt.Errorf("unproven block height is greater than current height")
	}

	k.unprovenBlocks[unprovenBlock.Height] = unprovenBlock.Block

	return nil
}
