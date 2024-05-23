package celestia

import (
	tmtypes "github.com/tendermint/tendermint/types"

	cometcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
)

// Convert celestia core' share proof to celestia light client's share proof
func TmShareProofToProto(sp *tmtypes.ShareProof) ShareProof {
	rowRoots := make([][]byte, len(sp.RowProof.RowRoots))
	rowProofs := make([]*cometcrypto.Proof, len(sp.RowProof.Proofs))

	for i := range sp.RowProof.RowRoots {
		rowRoots[i] = sp.RowProof.RowRoots[i].Bytes()
		rowProofs[i] = &cometcrypto.Proof{
			Total:    sp.RowProof.Proofs[i].Total,
			Index:    sp.RowProof.Proofs[i].Index,
			LeafHash: sp.RowProof.Proofs[i].LeafHash,
			Aunts:    sp.RowProof.Proofs[i].Aunts,
		}
	}

	shareProofs := make([]*NMTProof, len(sp.ShareProofs))
	for i := range sp.ShareProofs {
		shareProofs[i] = &NMTProof{
			Start:    sp.ShareProofs[i].Start,
			End:      sp.ShareProofs[i].End,
			Nodes:    sp.ShareProofs[i].Nodes,
			LeafHash: sp.ShareProofs[i].LeafHash,
		}
	}

	return ShareProof{
		Data:        sp.Data,
		ShareProofs: shareProofs,
		NamespaceId: sp.NamespaceID,
		RowProof: &RowProof{
			RowRoots: rowRoots,
			Proofs:   rowProofs,
			StartRow: sp.RowProof.StartRow,
			EndRow:   sp.RowProof.EndRow,
		},
		NamespaceVersion: sp.NamespaceVersion,
	}
}
