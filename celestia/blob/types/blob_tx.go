package types

import (
	appns "github.com/rollchains/tiablob/celestia/namespace"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type Blob = tmproto.Blob

// NewBlob creates a new coretypes.Blob from the provided data after performing
// basic stateless checks over it.
func NewBlob(ns appns.Namespace, blob []byte, shareVersion uint8) (*Blob, error) {
	err := ValidateBlobNamespace(ns)
	if err != nil {
		return nil, err
	}

	if len(blob) == 0 {
		return nil, ErrZeroBlobSize
	}

	return &tmproto.Blob{
		NamespaceId:      ns.ID,
		Data:             blob,
		ShareVersion:     uint32(shareVersion),
		NamespaceVersion: uint32(ns.Version),
	}, nil
}
