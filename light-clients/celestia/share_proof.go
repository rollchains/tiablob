package celestia

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"

	"github.com/celestiaorg/nmt"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"
)

type ShareProof struct {
	Data             [][]byte    `json:"data,omitempty"`
	ShareProofs      []*NMTProof `json:"share_proofs,omitempty"`
	NamespaceId      []byte      `json:"namespace_id,omitempty"`
	RowProof         *RowProof   `json:"row_proof,omitempty"`
	NamespaceVersion uint32      `json:"namespace_version,omitempty"`
}

// ShareProof is an NMT proof that a set of shares exist in a set of rows and a
// Merkle proof that those rows exist in a Merkle tree with a given data root.
type shareProof struct {
	// Data are the raw shares that are being proven.
	Data [][]byte `json:"data"`
	// ShareProofs are NMT proofs that the shares in Data exist in a set of
	// rows. There will be one ShareProof per row that the shares occupy.
	ShareProofs []*NMTProof `json:"share_proofs"`
	// NamespaceID is the namespace id of the shares being proven. This
	// namespace id is used when verifying the proof. If the namespace id doesn't
	// match the namespace of the shares, the proof will fail verification.
	NamespaceID      []byte   `json:"namespace_id"`
	RowProof         rowProof `json:"row_proof"`
	NamespaceVersion uint32   `json:"namespace_version"`
}

// NMTProof is a proof of a namespace.ID in an NMT.
// In case this proof proves the absence of a namespace.ID
// in a tree it also contains the leaf hashes of the range
// where that namespace would be.
type NMTProof struct {
	// Start index of this proof.
	Start int32 `json:"start,omitempty"`
	// End index of this proof.
	End int32 `json:"end,omitempty"`
	// Nodes that together with the corresponding leaf values can be used to
	// recompute the root and verify this proof. Nodes should consist of the max
	// and min namespaces along with the actual hash, resulting in each being 48
	// bytes each
	Nodes [][]byte `json:"nodes,omitempty"`
	// leafHash are nil if the namespace is present in the NMT. In case the
	// namespace to be proved is in the min/max range of the tree but absent, this
	// will contain the leaf hash necessary to verify the proof of absence. Leaf
	// hashes should consist of the namespace along with the actual hash,
	// resulting 40 bytes total.
	LeafHash []byte `json:"leaf_hash,omitempty"`
}

func (sp shareProof) ToProto() ShareProof {
	// TODO consider extracting a ToProto function for RowProof
	rowRoots := make([][]byte, len(sp.RowProof.RowRoots))
	rowProofs := make([]*crypto.Proof, len(sp.RowProof.Proofs))
	for i := range sp.RowProof.RowRoots {
		rowRoots[i] = sp.RowProof.RowRoots[i].Bytes()
		rowProofs[i] = sp.RowProof.Proofs[i].ToProto()
	}
	pbtp := ShareProof{
		Data:        sp.Data,
		ShareProofs: sp.ShareProofs,
		NamespaceId: sp.NamespaceID,
		RowProof: &RowProof{
			RowRoots: rowRoots,
			Proofs:   rowProofs,
			StartRow: sp.RowProof.StartRow,
			EndRow:   sp.RowProof.EndRow,
		},
		NamespaceVersion: sp.NamespaceVersion,
	}

	return pbtp
}

// shareProofFromProto creates a ShareProof from a proto message.
// Expects the proof to be pre-validated.
func shareProofFromProto(pb *ShareProof) (shareProof, error) {
	if pb == nil {
		return shareProof{}, fmt.Errorf("nil share proof protobuf")
	}

	return shareProof{
		RowProof:         rowProofFromProto(pb.RowProof),
		Data:             pb.Data,
		ShareProofs:      pb.ShareProofs,
		NamespaceID:      pb.NamespaceId,
		NamespaceVersion: pb.NamespaceVersion,
	}, nil
}

// Validate runs basic validations on the proof then verifies if it is consistent.
// It returns nil if the proof is valid. Otherwise, it returns a sensible error.
// The `root` is the block data root that the shares to be proven belong to.
// Note: these proofs are tested on the app side.
func (sp shareProof) Validate(root []byte) error {
	numberOfSharesInProofs := int32(0)
	for _, proof := range sp.ShareProofs {
		// the range is not inclusive from the left.
		numberOfSharesInProofs += proof.End - proof.Start
	}

	if len(sp.ShareProofs) != len(sp.RowProof.RowRoots) {
		return fmt.Errorf("the number of share proofs %d must equal the number of row roots %d", len(sp.ShareProofs), len(sp.RowProof.RowRoots))
	}
	if len(sp.Data) != int(numberOfSharesInProofs) {
		return fmt.Errorf("the number of shares %d must equal the number of shares in share proofs %d", len(sp.Data), numberOfSharesInProofs)
	}

	for _, proof := range sp.ShareProofs {
		if proof.Start < 0 {
			return errors.New("proof index cannot be negative")
		}
		if (proof.End - proof.Start) <= 0 {
			return errors.New("proof total must be positive")
		}
	}

	if err := sp.RowProof.Validate(root); err != nil {
		return err
	}

	if ok := sp.VerifyProof(); !ok {
		return errors.New("share proof failed to verify")
	}

	return nil
}

func (sp shareProof) VerifyProof() bool {
	cursor := int32(0)
	for i, proof := range sp.ShareProofs {
		nmtProof := nmt.NewInclusionProof(
			int(proof.Start),
			int(proof.End),
			proof.Nodes,
			true,
		)
		sharesUsed := proof.End - proof.Start
		if sp.NamespaceVersion > math.MaxUint8 {
			return false
		}
		// Consider extracting celestia-app's namespace package. We can't use it
		// here because that would introduce a circulcar import.
		namespace := append([]byte{uint8(sp.NamespaceVersion)}, sp.NamespaceID...)
		valid := nmtProof.VerifyInclusion(
			sha256.New(),
			namespace,
			sp.Data[cursor:sharesUsed+cursor],
			sp.RowProof.RowRoots[i],
		)
		if !valid {
			return false
		}
		cursor += sharesUsed
	}
	return true
}
