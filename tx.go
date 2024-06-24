package tiablob


import (
	"errors"
	
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	protov2 "google.golang.org/protobuf/proto"
)

var _ sdk.Tx = wInjectTx{}

type wInjectTx struct {
	Tx *InjectTx
	cdc codec.Codec
}

func NewInjectTx(cdc codec.Codec, msgs []sdk.Msg) wInjectTx {
	msgsAny := make([]*codectypes.Any, len(msgs))
	for i, msg := range msgs {
		msgAny, err := codectypes.NewAnyWithValue(msg)
		if err != nil {
			panic(err)
		}
		msgsAny[i] = msgAny
	}

	return wInjectTx{
		Tx: &InjectTx{
			Messages: msgsAny,
		},
		cdc: cdc,
	}
}

func (w wInjectTx) GetMsgs() []sdk.Msg {
	msgs, err := sdktx.GetMsgs(w.Tx.Messages, "tiablob.InjectTx")
	if err != nil {
		panic(err)
	}
	return msgs
}

func (w wInjectTx) GetMsgsV2() ([]protov2.Message, error) {
	msgsv2 := make([]protov2.Message, len(w.Tx.Messages))
	for i, msg := range w.Tx.Messages {
		_, msgv2, err := w.cdc.GetMsgAnySigners(msg)
		if err != nil {
			return nil, err
		}

		msgsv2[i] = msgv2
	}

	return msgsv2, nil

}

func TxDecoder(cdc codec.Codec, txDecoder sdk.TxDecoder) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		tx, err := txDecoder(txBytes)
		if err == nil {
			return tx, nil
		}

		itx := wInjectTx{
			cdc: cdc,
			Tx:  new(InjectTx),
		}
		if err2 := itx.Tx.Unmarshal(txBytes); err2 != nil {
			return nil, errors.Join(err, err2)
		}

		if err3 := sdktx.UnpackInterfaces(cdc.InterfaceRegistry(), itx.Tx.Messages); err3 != nil {
			return nil, errors.Join(err, err3)
		}

		return itx, nil
	}
}