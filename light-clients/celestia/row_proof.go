package celestia

import (
	"errors"
	"fmt"

	"github.com/cometbft/cometbft/crypto/merkle"
	crypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
)

type RowProof struct {
	RowRoots [][]byte        `protobuf:"bytes,1,rep,name=row_roots,json=rowRoots,proto3" json:"row_roots,omitempty"`
	Proofs   []*crypto.Proof `protobuf:"bytes,2,rep,name=proofs,proto3" json:"proofs,omitempty"`
	Root     []byte          `protobuf:"bytes,3,opt,name=root,proto3" json:"root,omitempty"`
	StartRow uint32          `protobuf:"varint,4,opt,name=start_row,json=startRow,proto3" json:"start_row,omitempty"`
	EndRow   uint32          `protobuf:"varint,5,opt,name=end_row,json=endRow,proto3" json:"end_row,omitempty"`
}

// RowProof is a Merkle proof that a set of rows exist in a Merkle tree with a
// given data root.
type rowProof struct {
	// RowRoots are the roots of the rows being proven.
	RowRoots []tmbytes.HexBytes `json:"row_roots"`
	// Proofs is a list of Merkle proofs where each proof proves that a row
	// exists in a Merkle tree with a given data root.
	Proofs   []*merkle.Proof `json:"proofs"`
	StartRow uint32          `json:"start_row"`
	EndRow   uint32          `json:"end_row"`
}

// Validate performs checks on the fields of this RowProof. Returns an error if
// the proof fails validation. If the proof passes validation, this function
// attempts to verify the proof. It returns nil if the proof is valid.
func (rp rowProof) Validate(root []byte) error {
	// HACKHACK performing subtraction with unsigned integers is unsafe.
	if int(rp.EndRow-rp.StartRow+1) != len(rp.RowRoots) {
		return fmt.Errorf("the number of rows %d must equal the number of row roots %d", int(rp.EndRow-rp.StartRow+1), len(rp.RowRoots))
	}
	if len(rp.Proofs) != len(rp.RowRoots) {
		return fmt.Errorf("the number of proofs %d must equal the number of row roots %d", len(rp.Proofs), len(rp.RowRoots))
	}
	if !rp.VerifyProof(root) {
		return errors.New("row proof failed to verify")
	}

	return nil
}

// VerifyProof verifies that all the row roots in this RowProof exist in a
// Merkle tree with the given root. Returns true if all proofs are valid.
func (rp rowProof) VerifyProof(root []byte) bool {
	for i, proof := range rp.Proofs {
		err := proof.Verify(root, rp.RowRoots[i])
		if err != nil {
			return false
		}
	}
	return true
}

func rowProofFromProto(p *RowProof) rowProof {
	if p == nil {
		return rowProof{}
	}
	rowRoots := make([]tmbytes.HexBytes, len(p.RowRoots))
	rowProofs := make([]*merkle.Proof, len(p.Proofs))
	for i := range p.Proofs {
		rowRoots[i] = p.RowRoots[i]
		rowProofs[i] = &merkle.Proof{
			Total:    p.Proofs[i].Total,
			Index:    p.Proofs[i].Index,
			LeafHash: p.Proofs[i].LeafHash,
			Aunts:    p.Proofs[i].Aunts,
		}
	}

	return rowProof{
		RowRoots: rowRoots,
		Proofs:   rowProofs,
		StartRow: p.StartRow,
		EndRow:   p.EndRow,
	}
}
