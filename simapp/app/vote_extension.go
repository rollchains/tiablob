package app

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ExtendVoteHandler(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return &abci.ResponseExtendVote{
		VoteExtension: []byte{},
	}, nil
}

func VerifyVoteExtensionHandler(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
}