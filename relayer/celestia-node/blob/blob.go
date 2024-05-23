package blob

import (
	//"bytes"
	"context"
	//"encoding/json"
	//"errors"
	//"fmt"

	"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/celestia-node/share"
	//"github.com/rollchains/tiablob/celestia/blob/types"
	//"github.com/celestiaorg/nmt"
	//appns "github.com/rollchains/tiablob/celestia/namespace"
	//"github.com/rollchains/tiablob/celestia/appconsts"
	//tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)
var _ Module = (*API)(nil)

// Module defines the API related to interacting with the blobs
//
//go:generate mockgen -destination=mocks/api.go -package=mocks . Module
type Module interface {
	// Submit sends Blobs and reports the height in which they were included.
	// Allows sending multiple Blobs atomically synchronously.
	// Uses default wallet registered on the Node.
	Submit(_ context.Context, _ []*blob.Blob, _ blob.GasPrice) (height uint64, _ error)
	// Get retrieves the blob by commitment under the given namespace and height.
	Get(_ context.Context, height uint64, _ share.Namespace, _ blob.Commitment) (*blob.Blob, error)
	// GetAll returns all blobs at the given height under the given namespaces.
	GetAll(_ context.Context, height uint64, _ []share.Namespace) ([]*blob.Blob, error)
	// GetProof retrieves proofs in the given namespaces at the given height by commitment.
	GetProof(_ context.Context, height uint64, _ share.Namespace, _ blob.Commitment) (*blob.Proof, error)
	// Included checks whether a blob's given commitment(Merkle subtree root) is included at
	// given height and under the namespace.
	Included(_ context.Context, height uint64, _ share.Namespace, _ *blob.Proof, _ blob.Commitment) (bool, error)
}

type API struct {
	Internal struct {
		Submit   func(context.Context, []*blob.Blob, blob.GasPrice) (uint64, error)                         `perm:"write"`
		Get      func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Blob, error)        `perm:"read"`
		GetAll   func(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)                     `perm:"read"`
		GetProof func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Proof, error)       `perm:"read"`
		Included func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) (bool, error) `perm:"read"`
	}
}

func (api *API) Submit(ctx context.Context, blobs []*blob.Blob, gasPrice blob.GasPrice) (uint64, error) {
	return api.Internal.Submit(ctx, blobs, gasPrice)
}

func (api *API) Get(
	ctx context.Context,
	height uint64,
	namespace share.Namespace,
	commitment blob.Commitment,
) (*blob.Blob, error) {
	return api.Internal.Get(ctx, height, namespace, commitment)
}

func (api *API) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return api.Internal.GetAll(ctx, height, namespaces)
}

func (api *API) GetProof(
	ctx context.Context,
	height uint64,
	namespace share.Namespace,
	commitment blob.Commitment,
) (*blob.Proof, error) {
	return api.Internal.GetProof(ctx, height, namespace, commitment)
}

func (api *API) Included(
	ctx context.Context,
	height uint64,
	namespace share.Namespace,
	proof *blob.Proof,
	commitment blob.Commitment,
) (bool, error) {
	return api.Internal.Included(ctx, height, namespace, proof, commitment)
}

/*var _ Module = (*API)(nil)

// Module defines the API related to interacting with the blobs
//
//go:generate mockgen -destination=mocks/api.go -package=mocks . Module
type Module interface {
	// Submit sends Blobs and reports the height in which they were included.
	// Allows sending multiple Blobs atomically synchronously.
	// Uses default wallet registered on the Node.
	Submit(_ context.Context, _ []*Blob, _ GasPrice) (height uint64, _ error)
	// Get retrieves the blob by commitment under the given namespace and height.
	Get(_ context.Context, height uint64, _ Namespace, _ Commitment) (*Blob, error)
	// GetAll returns all blobs at the given height under the given namespaces.
	GetAll(_ context.Context, height uint64, _ []Namespace) ([]*Blob, error)
	// GetProof retrieves proofs in the given namespaces at the given height by commitment.
	GetProof(_ context.Context, height uint64, _ Namespace, _ Commitment) (*Proof, error)
	// Included checks whether a blob's given commitment(Merkle subtree root) is included at
	// given height and under the namespace.
	Included(_ context.Context, height uint64, _ Namespace, _ *Proof, _ Commitment) (bool, error)
}

type API struct {
	Internal struct {
		Submit   func(context.Context, []*Blob, GasPrice) (uint64, error)                         `perm:"write"`
		Get      func(context.Context, uint64, Namespace, Commitment) (*Blob, error)        `perm:"read"`
		GetAll   func(context.Context, uint64, []Namespace) ([]*Blob, error)                     `perm:"read"`
		GetProof func(context.Context, uint64, Namespace, Commitment) (*Proof, error)       `perm:"read"`
		Included func(context.Context, uint64, Namespace, *Proof, Commitment) (bool, error) `perm:"read"`
	}
}

func (api *API) Submit(ctx context.Context, blobs []*Blob, gasPrice GasPrice) (uint64, error) {
	return api.Internal.Submit(ctx, blobs, gasPrice)
}

func (api *API) Get(
	ctx context.Context,
	height uint64,
	namespace Namespace,
	commitment Commitment,
) (*Blob, error) {
	return api.Internal.Get(ctx, height, namespace, commitment)
}

func (api *API) GetAll(ctx context.Context, height uint64, namespaces []Namespace) ([]*Blob, error) {
	return api.Internal.GetAll(ctx, height, namespaces)
}

func (api *API) GetProof(
	ctx context.Context,
	height uint64,
	namespace Namespace,
	commitment Commitment,
) (*Proof, error) {
	return api.Internal.GetProof(ctx, height, namespace, commitment)
}

func (api *API) Included(
	ctx context.Context,
	height uint64,
	namespace Namespace,
	proof *Proof,
	commitment Commitment,
) (bool, error) {
	return api.Internal.Included(ctx, height, namespace, proof, commitment)
}


// Blob represents any application-specific binary data that anyone can submit to Celestia.
type Blob struct {
	types.Blob `json:"blob"`

	Commitment Commitment `json:"commitment"`

	// the celestia-node's namespace type
	// this is to avoid converting to and from app's type
	namespace Namespace

	// index represents the index of the blob's first share in the EDS.
	// Only retrieved, on-chain blobs will have the index set. Default is -1.
	index int
}

// NewBlobV0 constructs a new blob from the provided Namespace and data.
// The blob will be formatted as v0 shares.
func NewBlobV0(namespace Namespace, data []byte) (*Blob, error) {
	return NewBlob(appconsts.ShareVersionZero, namespace, data)
}

// NewBlob constructs a new blob from the provided Namespace, data and share version.
func NewBlob(shareVersion uint8, namespace Namespace, data []byte) (*Blob, error) {
	if len(data) == 0 || len(data) > appconsts.DefaultMaxBytes {
		return nil, fmt.Errorf("blob data must be > 0 && <= %d, but it was %d bytes", appconsts.DefaultMaxBytes, len(data))
	}
	if err := namespace.ValidateForBlob(); err != nil {
		return nil, err
	}

	blob := tmproto.Blob{
		NamespaceId:      namespace.ID(),
		Data:             data,
		ShareVersion:     uint32(shareVersion),
		NamespaceVersion: uint32(namespace.Version()),
	}

	com, err := types.CreateCommitment(&blob)
	if err != nil {
		return nil, err
	}
	return &Blob{Blob: blob, Commitment: com, namespace: namespace, index: -1}, nil
}

// Namespace returns blob's namespace.
func (b *Blob) Namespace() Namespace {
	return b.namespace
}

// Index returns the blob's first share index in the EDS.
// Only retrieved, on-chain blobs will have the index set. Default is -1.
func (b *Blob) Index() int {
	return b.index
}

func (b *Blob) compareCommitments(com Commitment) bool {
	return bytes.Equal(b.Commitment, com)
}

type jsonBlob struct {
	Namespace    Namespace `json:"namespace"`
	Data         []byte          `json:"data"`
	ShareVersion uint32          `json:"share_version"`
	Commitment   Commitment      `json:"commitment"`
	Index        int             `json:"index"`
}

func (b *Blob) MarshalJSON() ([]byte, error) {
	blob := &jsonBlob{
		Namespace:    b.Namespace(),
		Data:         b.Data,
		ShareVersion: b.ShareVersion,
		Commitment:   b.Commitment,
		Index:        b.index,
	}
	return json.Marshal(blob)
}

func (b *Blob) UnmarshalJSON(data []byte) error {
	var blob jsonBlob
	err := json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	b.Blob.NamespaceVersion = uint32(blob.Namespace.Version())
	b.Blob.NamespaceId = blob.Namespace.ID()
	b.Blob.Data = blob.Data
	b.Blob.ShareVersion = blob.ShareVersion
	b.Commitment = blob.Commitment
	b.namespace = blob.Namespace
	b.index = blob.Index
	return nil
}


// Commitment is a Merkle Root of the subtree built from shares of the Blob.
// It is computed by splitting the blob into shares and building the Merkle subtree to be included
// after Submit.
type Commitment []byte

func (com Commitment) String() string {
	return string(com)
}

// Equal ensures that commitments are the same
func (com Commitment) Equal(c Commitment) bool {
	return bytes.Equal(com, c)
}

// Proof is a collection of nmt.Proofs that verifies the inclusion of the data.
// Proof proves the WHOLE namespaced data for the particular row.
// TODO (@vgonkivs): rework `Proof` in order to prove a particular blob.
// https://github.com/celestiaorg/celestia-node/issues/2303
type Proof []*nmt.Proof

func (p Proof) Len() int { return len(p) }

// equal is a temporary method that compares two proofs.
// should be removed in BlobService V1.
func (p Proof) equal(input Proof) error {
	if p.Len() != input.Len() {
		return ErrInvalidProof
	}

	for i, proof := range p {
		pNodes := proof.Nodes()
		inputNodes := input[i].Nodes()
		for i, node := range pNodes {
			if !bytes.Equal(node, inputNodes[i]) {
				return ErrInvalidProof
			}
		}

		if proof.Start() != input[i].Start() || proof.End() != input[i].End() {
			return ErrInvalidProof
		}

		if !bytes.Equal(proof.LeafHash(), input[i].LeafHash()) {
			return ErrInvalidProof
		}

	}
	return nil
}
// GasPrice represents the amount to be paid per gas unit. Fee is set by
// multiplying GasPrice by GasLimit, which is determined by the blob sizes.
type GasPrice float64

// Namespace represents namespace of a Share.
// Consists of version byte and namespace ID.
type Namespace []byte

// Version reports version of the Namespace.
func (n Namespace) Version() byte {
	return n[appns.NamespaceVersionSize-1]
}

// ValidateForBlob checks if the Namespace is valid blob namespace.
func (n Namespace) ValidateForBlob() error {
	return nil
}

type ID []byte

// ID reports ID of the Namespace.
func (n Namespace) ID() ID {
	return ID(n[appns.NamespaceVersionSize:])
}*/
