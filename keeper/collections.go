package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/rollchains/tiablob"
)

const (
	// Window for a transaction to be committed on Celestia
	CelestiaResubmissionTime = 75 * time.Second

	// Buffer for relayer polling logic to retrieve a proof
	RelayerPollingBuffer = 15 * time.Second
)

func (k *Keeper) SetValidatorCelestiaAddress(ctx context.Context, validator tiablob.Validator) error {
	return k.Validators.Set(ctx, validator.ValidatorAddress, validator.CelestiaAddress)
}

func (k *Keeper) GetValidatorCelestiaAddress(ctx context.Context, validatorAddress string) (string, error) {
	return k.Validators.Get(ctx, validatorAddress)
}

func (k *Keeper) GetAllValidators(ctx context.Context) (tiablob.Validators, error) {
	var validators tiablob.Validators
	it, err := k.Validators.Iterate(ctx, nil)
	if err != nil {
		return validators, err
	}

	defer it.Close()

	for ; it.Valid(); it.Next() {
		var validator tiablob.Validator
		validator.ValidatorAddress, err = it.Key()
		if err != nil {
			return validators, err
		}
		validator.CelestiaAddress, err = it.Value()
		if err != nil {
			return validators, err
		}
		validators.Validators = append(validators.Validators, validator)
	}

	return validators, nil
}

func (k *Keeper) SetProvenHeight(ctx context.Context, height int64) error {
	return k.ProvenHeight.Set(ctx, height)
}

func (k *Keeper) GetProvenHeight(ctx context.Context) (int64, error) {
	return k.ProvenHeight.Get(ctx)
}

func (k *Keeper) SetClientID(ctx context.Context, clientID string) error {
	return k.ClientID.Set(ctx, clientID)
}

func (k *Keeper) GetClientID(ctx context.Context) (string, error) {
	return k.ClientID.Get(ctx)
}

// IsBlockPending return true if a block height is already pending
func (k Keeper) IsBlockPending(ctx context.Context, blockHeight int64) bool {
	found, err := k.PendingBlocksToTimeouts.Has(ctx, blockHeight)
	if err != nil {
		return false
	}
	return found
}

// IsBlockExpired will return true if a block is pending and expired, otherwise it returns false
func (k *Keeper) IsBlockExpired(ctx context.Context, currentBlockTime time.Time, blockHeight int64) bool {
	currentBlockTimeNs := currentBlockTime.UnixNano()
	found, err := k.PendingBlocksToTimeouts.Has(ctx, blockHeight)
	if err != nil {
		return false
	}
	if found {
		expiration, err := k.PendingBlocksToTimeouts.Get(ctx, blockHeight)
		if err != nil {
			return false
		}
		if currentBlockTimeNs >= expiration {
			return true
		}
	}
	return false
}

// AddUpdatePendingBlock will add a new pending block or update an existing pending block
func (k *Keeper) AddUpdatePendingBlock(ctx context.Context, cdc codec.BinaryCodec, pendingBlock int64, currentBlockTime time.Time) error {
	found, err := k.PendingBlocksToTimeouts.Has(ctx, pendingBlock)
	if err != nil {
		return fmt.Errorf("remove pending blocks, block %d error", pendingBlock)
	}
	if found {
		if err = k.RemovePendingBlock(ctx, cdc, pendingBlock); err != nil {
			return err
		}
	}
	expiration := currentBlockTime.Add(CelestiaResubmissionTime + RelayerPollingBuffer).UnixNano()
	if err = k.PendingBlocksToTimeouts.Set(ctx, pendingBlock, expiration); err != nil {
		return fmt.Errorf("add/update pending block, set pending block (%d) to timeout (%d)", pendingBlock, expiration)
	}
	found, err = k.TimeoutsToPendingBlocks.Has(ctx, expiration)
	var pendingBlocks PendingBlocks
	if found {
		pendingBlocks, err = k.TimeoutsToPendingBlocks.Get(ctx, expiration)
		if err != nil {
			return fmt.Errorf("add/update pending block, unmarshal existing pending blocks")
		}
	}
	pendingBlocks.BlockHeights = append(pendingBlocks.BlockHeights, pendingBlock)
	if err = k.TimeoutsToPendingBlocks.Set(ctx, expiration, pendingBlocks); err != nil {
		return fmt.Errorf("add/update pending block, set new timeouts to pending blocks")
	}
	return nil
}

// RemovePendingBlock removes proven block from pending state
// This function will remove the proven block from the PendingBlocksToTimeouts map and TimeoutsToPendingBlocks map
func (k *Keeper) RemovePendingBlock(ctx context.Context, cdc codec.BinaryCodec, provenBlock int64) error {
	found, err := k.PendingBlocksToTimeouts.Has(ctx, provenBlock)
	if err != nil {
		return fmt.Errorf("remove pending blocks, block %d error", provenBlock)
	}
	if found {
		expiration, err := k.PendingBlocksToTimeouts.Get(ctx, provenBlock)
		if err != nil {
			return fmt.Errorf("remove pending blocks, getting pending block %d", provenBlock)
		}
		if err = k.PendingBlocksToTimeouts.Remove(ctx, provenBlock); err != nil {
			return fmt.Errorf("remove pending blocks, removing block %d", provenBlock)
		}
		pendingBlocks, err := k.TimeoutsToPendingBlocks.Get(ctx, expiration)
		if err != nil {
			return fmt.Errorf("remove pending blocks, getting expiration %d", expiration)
		}
		var newPendingBlocks []int64
		for _, blockHeight := range pendingBlocks.BlockHeights {
			if blockHeight != provenBlock {
				newPendingBlocks = append(newPendingBlocks, blockHeight)
			}
		}
		if len(newPendingBlocks) > 0 {
			pendingBlocks.BlockHeights = newPendingBlocks
			if err = k.TimeoutsToPendingBlocks.Set(ctx, expiration, pendingBlocks); err != nil {
				return fmt.Errorf("remove pending block, set new pending blocks")
			}
		} else {
			if err = k.TimeoutsToPendingBlocks.Remove(ctx, expiration); err != nil {
				return fmt.Errorf("remove pending blocks, removing timeout set %d", expiration)
			}
		}
	}
	return nil
}

// GetExpiredBlocks returns all expired blocks, proposer will propose publishing based on this set
func (k Keeper) GetExpiredBlocks(ctx context.Context, cdc codec.BinaryCodec, currentBlockTime time.Time) []int64 {
	currentBlockTimeNs := currentBlockTime.UnixNano()
	iterator, err := k.TimeoutsToPendingBlocks.
		Iterate(ctx, (&collections.Range[int64]{}).StartInclusive(0).EndInclusive(currentBlockTimeNs))
	if err != nil {
		return nil
	}
	defer iterator.Close()

	var expiredBlocks []int64
	for ; iterator.Valid(); iterator.Next() {
		pendingBlocks, err := iterator.Value()
		if err != nil {
			return nil
		}
		expiredBlocks = append(expiredBlocks, pendingBlocks.BlockHeights...)
	}
	return expiredBlocks
}
