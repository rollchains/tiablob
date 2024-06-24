package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	
	"github.com/rollchains/tiablob"
)

type msgServer struct {
	*Keeper
}

var (
	_ tiablob.MsgServer = msgServer{}
	_, _ sdk.Msg = &tiablob.MsgSetCelestiaAddress{}, &tiablob.MsgInjectedData{}
)

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper *Keeper) tiablob.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (s msgServer) SetCelestiaAddress(ctx context.Context, msg *tiablob.MsgSetCelestiaAddress) (*tiablob.MsgSetCelestiaAddressResponse, error) {
	valAddr, err := msg.Validate(s.Keeper.stakingKeeper.ValidatorAddressCodec())
	if err != nil {
		return nil, err
	}

	// verify that the validator exists
	if _, err := s.Keeper.stakingKeeper.GetValidator(ctx, valAddr); err != nil {
		return nil, err
	}

	if err = s.Keeper.SetValidatorCelestiaAddress(ctx, tiablob.Validator{
		ValidatorAddress: msg.ValidatorAddress,
		CelestiaAddress:  msg.CelestiaAddress,
	}); err != nil {
		return nil, err
	}

	return new(tiablob.MsgSetCelestiaAddressResponse), nil
}

func (s msgServer) InjectedData(goCtx context.Context, msg *tiablob.MsgInjectedData) (*tiablob.MsgInjectedDataResponse, error) {
	fmt.Println("In ExecuteInjectedData")
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := s.PreblockerCreateClient(ctx, msg.CreateClient); err != nil {
		return nil, err
	}
	if err := s.PreblockerHeaders(ctx, msg.Headers); err != nil {
		return nil, err
	}
	if err := s.PreblockerProofs(ctx, msg.Proofs); err != nil {
		return nil, err
	}
	if err := s.PreblockerPendingBlocks(ctx, msg.BlockTime, msg.ProposerAddress, &msg.PendingBlocks); err != nil {
		return nil, err
	}
	fmt.Println("No error")

	return &tiablob.MsgInjectedDataResponse{}, nil
}