package tiablob

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type InjectedTxDecorator struct {}

func NewInjectedTxDecorator() InjectedTxDecorator {
	return InjectedTxDecorator{}
}

func (itd InjectedTxDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if _, ok := tx.(wInjectTx); !ok {
		return next(ctx, tx, simulate)
	}

	// only allow if we are in deliver tx (only way is via proposer injected tx)
	if ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		fmt.Println("wInjectTx antehandle check, recheck, simulate", ctx.IsCheckTx(), ctx.IsReCheckTx(), simulate)
		return ctx, ErrTxNotAllowed
	}

	msgs := tx.GetMsgs()

	// make sure entire tx is only allowed msgs
	for _, m := range msgs {
		switch m.(type) {
		case *MsgInjectedData:

		default:
			return ctx, ErrTiablobActionNotAllowed
		}
	}

	// skip rest of antes
	return ctx, nil
}
