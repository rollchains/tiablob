package keeper

import (
	"context"

	"github.com/rollchains/tiablob"
)

type msgServer struct {
	k *Keeper
}

var _ tiablob.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper *Keeper) tiablob.MsgServer {
	return &msgServer{k: keeper}
}

func (s msgServer) SetCelestiaAddress(ctx context.Context, msg *tiablob.MsgSetCelestiaAddress) (*tiablob.MsgSetCelestiaAddressResponse, error) {
	valAddr, err := msg.Validate(s.k.stakingKeeper.ValidatorAddressCodec())
	if err != nil {
		return nil, err
	}

	// verify that the validator exists
	if _, err := s.k.stakingKeeper.GetValidator(ctx, valAddr); err != nil {
		return nil, err
	}

	s.k.SetValidatorCelestiaAddress(ctx, tiablob.Validator{
		ValidatorAddress: msg.ValidatorAddress,
		CelestiaAddress:  msg.CelestiaAddress,
	})

	return new(tiablob.MsgSetCelestiaAddressResponse), nil
}

/*func (s msgServer) ProveBlock(ctx context.Context, msg *tiablob.MsgProveBlock) (*tiablob.MsgProveBlockResponse, error) {
	valAddr, err := s.k.stakingKeeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	// verify that the validator exists
	if _, err := s.k.stakingKeeper.GetValidator(ctx, valAddr); err != nil {
		return nil, fmt.Errorf("validator %s not found: %w", valAddr, err)
	}

	provenHeight, err := s.k.GetProvenHeight(ctx)
	if err != nil {
		return nil, err
	}

	if msg.Height != provenHeight+1 {
		return nil, sdkerrors.ErrInvalidHeight.Wrapf("next prove block height must be %d", provenHeight+1)
	}

	// clientID, err := s.k.GetClientID(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// TODO verify block proof against the light client

	if err := s.k.SetProvenHeight(ctx, msg.Height); err != nil {
		return nil, err
	}

	// Notify the relayer that a new block has been proven
	s.k.relayer.NotifyProvenHeight(msg.Height)

	return new(tiablob.MsgProveBlockResponse), nil
}*/
