package celestia

import (
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	tmsecp256k1 "github.com/tendermint/tendermint/crypto/secp256k1"
	tmtypes "github.com/tendermint/tendermint/types"

	//tmsr25519 "github.com/tendermint/tendermint/crypto/sr25519"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/crypto/sr25519"
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/types"
)

func tm2CometLightBlock(light *tmtypes.LightBlock) *types.LightBlock {
	var signatures []types.CommitSig
	for _, sig := range light.SignedHeader.Commit.Signatures {
		signatures = append(signatures, types.CommitSig{
			BlockIDFlag:      types.BlockIDFlag(sig.BlockIDFlag),
			ValidatorAddress: bytes.HexBytes(sig.ValidatorAddress),
			Timestamp:        sig.Timestamp,
			Signature:        sig.Signature,
		})
	}

	var validators []*types.Validator
	for _, validator := range light.ValidatorSet.Validators {
		validators = append(validators, &types.Validator{
			Address:          bytes.HexBytes(validator.Address),
			PubKey:           convertPubKey(validator.PubKey),
			VotingPower:      validator.VotingPower,
			ProposerPriority: validator.ProposerPriority,
		})
	}

	return &types.LightBlock{
		SignedHeader: &types.SignedHeader{
			Header: &types.Header{
				Version: version.Consensus{
					Block: light.SignedHeader.Header.Version.Block,
					App:   light.SignedHeader.Header.Version.App,
				},
				ChainID: light.SignedHeader.Header.ChainID,
				Height:  light.SignedHeader.Header.Height,
				Time:    light.SignedHeader.Header.Time,
				LastBlockID: types.BlockID{
					Hash: bytes.HexBytes(light.SignedHeader.Header.LastBlockID.Hash),
					PartSetHeader: types.PartSetHeader{
						Total: light.SignedHeader.Header.LastBlockID.PartSetHeader.Total,
						Hash:  bytes.HexBytes(light.SignedHeader.Header.LastBlockID.PartSetHeader.Hash),
					},
				},
				LastCommitHash:     bytes.HexBytes(light.SignedHeader.Header.LastCommitHash),
				DataHash:           bytes.HexBytes(light.SignedHeader.Header.DataHash),
				ValidatorsHash:     bytes.HexBytes(light.SignedHeader.Header.ValidatorsHash),
				NextValidatorsHash: bytes.HexBytes(light.SignedHeader.Header.NextValidatorsHash),
				ConsensusHash:      bytes.HexBytes(light.SignedHeader.Header.ConsensusHash),
				AppHash:            bytes.HexBytes(light.SignedHeader.Header.AppHash),
				LastResultsHash:    bytes.HexBytes(light.SignedHeader.Header.LastResultsHash),
				EvidenceHash:       bytes.HexBytes(light.SignedHeader.Header.EvidenceHash),
				ProposerAddress:    bytes.HexBytes(light.SignedHeader.Header.ProposerAddress),
			},
			Commit: &types.Commit{
				Height: light.SignedHeader.Commit.Height,
				Round:  light.SignedHeader.Commit.Round,
				BlockID: types.BlockID{
					Hash: bytes.HexBytes(light.SignedHeader.Commit.BlockID.Hash),
					PartSetHeader: types.PartSetHeader{
						Total: light.SignedHeader.Commit.BlockID.PartSetHeader.Total,
						Hash:  bytes.HexBytes(light.SignedHeader.Commit.BlockID.PartSetHeader.Hash),
					},
				},
				Signatures: signatures,
			},
		},
		ValidatorSet: &types.ValidatorSet{
			Validators: validators,
			Proposer: &types.Validator{
				Address:          bytes.HexBytes(light.ValidatorSet.Proposer.Address),
				PubKey:           convertPubKey(light.ValidatorSet.Proposer.PubKey),
				VotingPower:      light.ValidatorSet.Proposer.VotingPower,
				ProposerPriority: light.ValidatorSet.Proposer.ProposerPriority,
			},
		},
	}
}

func convertPubKey(pk tmcrypto.PubKey) crypto.PubKey {
	switch key := pk.(type) {
	case tmed25519.PubKey:
		return ed25519.PubKey(key)
	case tmsecp256k1.PubKey:
		return secp256k1.PubKey(key)
	// go-schnorrkel dependency conflict between celestia-core and interchaintest prevents tmsr25519 from being included
	//case tmsr25519.PubKey:
	//	return sr25519.PubKey(key)
	default:
		return sr25519.PubKey(pk.Bytes())
		//return nil
	}
}
