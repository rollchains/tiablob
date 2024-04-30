package celestia

import (
	nodeblob "github.com/rollchains/tiablob/celestia-node/blob"
	nodeshare "github.com/rollchains/tiablob/celestia-node/share"
	appproto "github.com/rollchains/tiablob/celestia/blob"

	tmtypes "github.com/tendermint/tendermint/types"

	cometcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
)

// Just json marshal from one type, unmarshal to the other
// TODO: optimize this
func TmShareProofToProto(sp *tmtypes.ShareProof) ShareProof {
	/*spBz, err := json.Marshal(sp)
	if err != nil {
		return nil, err
	}

	var shareProof ShareProof
	if err = json.Unmarshal(spBz, &shareProof); err != nil {
		return nil, err
	}*/

	rowRoots := make([][]byte, len(sp.RowProof.RowRoots))
	rowProofs := make([]*cometcrypto.Proof, len(sp.RowProof.Proofs))

	for i := range sp.RowProof.RowRoots {
		rowRoots[i] = sp.RowProof.RowRoots[i].Bytes()
		rowProofs[i] = &cometcrypto.Proof{
			Total: sp.RowProof.Proofs[i].Total,
			Index: sp.RowProof.Proofs[i].Index,
			LeafHash: sp.RowProof.Proofs[i].LeafHash,
			Aunts: sp.RowProof.Proofs[i].Aunts,
		}
	}

	shareProofs := make([]*NMTProof, len(sp.ShareProofs))
	for i := range sp.ShareProofs {
		shareProofs[i] = &NMTProof{
			Start: sp.ShareProofs[i].Start,
			End: sp.ShareProofs[i].End,
			Nodes: sp.ShareProofs[i].Nodes,
			LeafHash: sp.ShareProofs[i].LeafHash,
		}
	}

	return ShareProof{
		Data: sp.Data,
		ShareProofs: shareProofs,
		NamespaceId: sp.NamespaceID,
		RowProof: &RowProof{
			RowRoots: rowRoots,
			Proofs: rowProofs,
			StartRow: sp.RowProof.StartRow,
			EndRow: sp.RowProof.EndRow,
		},
		NamespaceVersion: sp.NamespaceVersion,
	}
}

func BlobToProto(blob *nodeblob.Blob) appproto.Blob {
	return appproto.Blob{
		NamespaceId: blob.NamespaceId,
		Data: blob.Data,
		ShareVersion: blob.ShareVersion,
		NamespaceVersion: blob.NamespaceVersion,
	}
}

func BlobFromProto(blobProto *appproto.Blob) (*nodeblob.Blob, error) {
	namespace, err := nodeshare.NewBlobNamespaceV0(blobProto.NamespaceId[18:])
	if err != nil {
		return nil, err
	}
	return nodeblob.NewBlobV0(namespace, blobProto.Data)
}