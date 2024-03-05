package tiablob

import (
	fmt "fmt"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgSetCelestiaAddress) Validate(ac address.Codec) ([]byte, error) {
	valAddr, err := ac.StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	if msg.CelestiaAddress == "" {
		return nil, fmt.Errorf("validator celestia address cannot be empty")
	}

	hrp, addrBz, err := bech32.DecodeAndConvert(msg.CelestiaAddress)
	if err != nil {
		return nil, fmt.Errorf("decoding celestia address failed: %w", err)
	}

	if hrp != Bech32Celestia {
		return nil, fmt.Errorf("invalid celestia address prefix: %s", hrp)
	}

	if len(addrBz) != AddrLen {
		return nil, fmt.Errorf("invalid celestia address length: %d", len(addrBz))
	}

	return valAddr, nil
}
