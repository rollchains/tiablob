package app

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ExtendVoteHandler(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	fmt.Println("ExtendVoteHandler, height:", req.Height)
	return &abci.ResponseExtendVote{
		VoteExtension: []byte{1},
	}, nil
}

func VerifyVoteExtensionHandler(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	fmt.Println("VerifyVoteExtensionHandler", req.Height, string(req.ValidatorAddress))
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
}