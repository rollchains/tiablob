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

	if err = s.k.SetValidatorCelestiaAddress(ctx, tiablob.Validator{
		ValidatorAddress: msg.ValidatorAddress,
		CelestiaAddress:  msg.CelestiaAddress,
	}); err != nil {
		return nil, err
	}

	return new(tiablob.MsgSetCelestiaAddressResponse), nil
}
