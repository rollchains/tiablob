package store

import (
	cmtmath "github.com/cometbft/cometbft/libs/math"
)

const (
	// BlockPartSizeBytes is the size of one block part.
	BlockPartSizeBytes uint32 = 65536 // 64kB
)

type PartSet struct {
	total uint32
	parts [][]byte
}

// Returns an immutable, full PartSet from the data bytes.
// The data bytes are split into "partSize" chunks, and merkle tree computed.
// CONTRACT: partSize is greater than zero.
func NewPartSetFromData(data []byte, partSize uint32) *PartSet {
	// divide data into 4kb parts.
	total := (uint32(len(data)) + partSize - 1) / partSize
	parts := make([][]byte, total)
	for i := uint32(0); i < total; i++ {
		part := data[i*partSize : cmtmath.MinInt(len(data), int((i+1)*partSize))]
		parts[i] = part
	}
	
	return &PartSet{
		total:         total,
		parts:         parts,
	}
}

func (p *PartSet) Total() uint32 {
	return p.total
}

func (p *PartSet) GetPart(i int) []byte {
	return p.parts[i]
}