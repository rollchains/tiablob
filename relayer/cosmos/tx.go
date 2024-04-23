package cosmos

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

func (cc *CosmosProvider) SignAndBroadcast(
	ctx context.Context,
	chainID string,
	gasPrices string,
	gasAdjustment float64,
	gasOverride uint64,
	bech32Prefix string,
	keyName string,
	feeGranter sdk.AccAddress,
	msgs []sdk.Msg,
	memo string,
) (*coretypes.ResultBroadcastTxCommit, error) {
	walletState := cc.EnsureWalletState(keyName)

	sequence, txBz, err := cc.Sign(ctx, walletState, chainID, gasPrices,
		gasAdjustment, gasOverride, bech32Prefix, keyName, feeGranter, msgs, memo)
	if err != nil {
		return nil, err
	}

	return cc.Broadcast(ctx, sequence, walletState, txBz)
}

func (cc *CosmosProvider) Sign(
	ctx context.Context,
	walletState *WalletState,
	chainID string,
	gasPrices string,
	gasAdjustment float64,
	gasOverride uint64,
	bech32Prefix string,
	keyName string,
	feeGranter sdk.AccAddress,
	msgs []sdk.Msg,
	memo string,
) (uint64, []byte, error) {
	var txf tx.Factory

	acc, err := cc.ShowAddress(keyName, bech32Prefix)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get key address for key %s: %w", keyName, err)
	}

	ac, err := cc.AccountInfo(ctx, acc)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get account info for address %s: %w", acc, err)
	}

	walletState.updateNextAccountSequence(ac.GetSequence())

	txf = txf.
		WithKeybase(cc.keybase).
		WithTxConfig(cc.cdc.TxConfig).
		WithSignMode(signing.SignMode(cc.cdc.TxConfig.SignModeHandler().DefaultMode())).
		WithChainID(chainID).
		WithGasPrices(gasPrices).
		WithAccountNumber(ac.GetAccountNumber()).
		WithSequence(walletState.NextAccountSequence).
		WithMemo(memo).
		WithBech32Prefix(bech32Prefix)

	if feeGranter != nil {
		txf = txf.WithFeeGranter(feeGranter)
	}

	keyInfo, err := cc.keybase.Key(keyName)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get key info for key %s: %w", keyName, err)
	}

	pubKey, err := keyInfo.GetPubKey()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get pubkey for key %s: %w", keyName, err)
	}

	if gasOverride > 0 {
		txf = txf.WithGas(gasOverride)
	} else {
		gas, err := cc.EstimateGas(ctx, txf, pubKey, msgs...)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to estimate gas: %w", err)
		}

		gas = uint64(float64(gas) * gasAdjustment)

		txf = txf.WithGas(gas)
	}

	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	if err := tx.Sign(ctx, txf, keyName, txb, false); err != nil {
		return 0, nil, err
	}

	txBz, err := cc.cdc.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return 0, nil, fmt.Errorf("failed to encode tx for broadcast: %w", err)
	}

	return txf.Sequence(), txBz, nil
}

func (cc *CosmosProvider) Broadcast(
	ctx context.Context,
	sequence uint64,
	walletState *WalletState,
	txBz []byte,
) (*coretypes.ResultBroadcastTxCommit, error) {
	// Broadcast the transaction.
	res, err := cc.rpcClient.BroadcastTxCommit(ctx, txBz)
	if err != nil {
		return nil, err
	}

	if res.CheckTx.Code != 0 {
		return res, fmt.Errorf("failed to broadcast, CheckTx failed: %d - %s - %s - %s", res.CheckTx.Code, res.CheckTx.Codespace, res.CheckTx.Log, string(res.CheckTx.Data))
	}

	if res.TxResult.Code != 0 {
		return res, fmt.Errorf("failed to broadcast, Tx execution failed: %d - %s - %s - %s", res.TxResult.Code, res.TxResult.Codespace, res.TxResult.Log, string(res.TxResult.Data))
	}

	walletState.updateNextAccountSequence(sequence + 1)

	return res, nil
}

func (cc *CosmosProvider) EncodeTxJSON(txb client.TxBuilder) ([]byte, error) {
	return cc.cdc.TxConfig.TxJSONEncoder()(txb.GetTx())
}

// EstimateGas simulates a tx to generate the appropriate gas settings before broadcasting a tx.
func (cc *CosmosProvider) EstimateGas(ctx context.Context, txf tx.Factory, pubKey cryptotypes.PubKey, msgs ...sdk.Msg) (uint64, error) {
	txBytes, err := BuildSimTx(pubKey, txf, msgs...)
	if err != nil {
		return 0, err
	}

	simQuery := abci.RequestQuery{
		Path: "/cosmos.tx.v1beta1.Service/Simulate",
		Data: txBytes,
	}

	res, err := cc.QueryABCI(ctx, simQuery)
	if err != nil {
		return 0, err
	}

	var simRes txtypes.SimulateResponse
	if err := simRes.Unmarshal(res.Value); err != nil {
		return 0, err
	}

	return simRes.GasInfo.GasUsed, err
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be built.
func BuildSimTx(pubKey cryptotypes.PubKey, txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode: txf.SignMode(),
		},
		Sequence: txf.Sequence(),
	}
	if err := txb.SetSignatures(sig); err != nil {
		return nil, err
	}

	protoProvider, ok := txb.(protoTxProvider)
	if !ok {
		return nil, fmt.Errorf("cannot simulate amino tx")
	}

	simReq := txtypes.SimulateRequest{Tx: protoProvider.GetProtoTx()}
	return simReq.Marshal()
}

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}
