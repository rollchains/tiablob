package store

import (
	cmtmath "github.com/cometbft/cometbft/libs/math"
)

const (
	// BlockPartSizeBytes is the size of one block part.
	BlockPartSizeBytes = 65536 // 64kB
)

type PartSet struct {
	total int
	parts [][]byte
}

func NewPartSetFromData(data []byte, partSize int) *PartSet {
	// divide data into 4kb parts.
	total := (len(data) + partSize - 1) / partSize
	parts := make([][]byte, total)
	for i := 0; i < total; i++ {
		part := data[i*partSize : cmtmath.MinInt(len(data), (i+1)*partSize)]
		parts[i] = part
	}

	return &PartSet{
		total: total,
		parts: parts,
	}
}

func (p *PartSet) Total() int {
	return p.total
}

func (p *PartSet) GetPart(i int) []byte {
	return p.parts[i]
}
