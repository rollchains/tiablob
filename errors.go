package tiablob

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrTiablobActionNotAllowed = sdkerrors.Register(ModuleName, 1, "this action is not allowed via a tiablob injected transaction")
	ErrTxNotAllowed            = sdkerrors.Register(ModuleName, 2, "this transaction is only allowed via a tiablob injected transaction")
)
