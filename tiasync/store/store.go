package store

import (
	"encoding/binary"
	//"encoding/hex"
	"encoding/json"
	//"errors"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	dbm "github.com/cometbft/cometbft-db"

	//"github.com/cometbft/cometbft/evidence"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	//cmtstore "github.com/cometbft/cometbft/proto/tendermint/store"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	//sm "github.com/cometbft/cometbft/state"
	//"github.com/cometbft/cometbft/types"
)

// Assuming the length of a block part is 64kB (`types.BlockPartSizeBytes`),
// the maximum size of a block, that will be batch saved, is 640kB. The
// benchmarks have shown that `goleveldb` still performs well with blocks of
// this size. However, if the block is larger than 1MB, the performance degrades.
const maxBlockPartsToBatch = 10

/*
BlockStore is a simple low level store for blocks.

There are three types of information stored:
  - BlockMeta:   Meta information about each block
  - Block part:  Parts of each block, aggregated w/ PartSet
  - Commit:      The commit part of each block, for gossiping precommit votes

Currently the precommit signatures are duplicated in the Block parts as
well as the Commit.  In the future this may change, perhaps by moving
the Commit data outside the Block. (TODO)

The store can be assumed to contain all contiguous blocks between base and height (inclusive).

// NOTE: BlockStore methods will panic if they encounter errors
// deserializing loaded data, indicating probable corruption on disk.
*/
type BlockStore struct {
	db dbm.DB

	// mtx guards access to the struct fields listed below it. Although we rely on the database
	// to enforce fine-grained concurrency control for its data, we need to make sure that
	// no external observer can get data from the database that is not in sync with the fields below,
	// and vice-versa. Hence, when updating the fields below, we use the mutex to make sure
	// that the database is also up to date. This prevents any concurrent external access from
	// obtaining inconsistent data.
	// The only reason for keeping these fields in the struct is that the data
	// can't efficiently be queried from the database since the key encoding we use is not
	// lexicographically ordered (see https://github.com/tendermint/tendermint/issues/4567).
	mtx    cmtsync.RWMutex
	base   int64
	height int64
	lastCelestiaHeightQueried int64
}

// TODO: make proto
type BlockStoreState struct {
	Base int64 `json:"base"`
	Height int64 `json:"height"`
	LastCelestiaHeightQueried int64 `json:"last_celestia_height_queried"`
}

// NewBlockStore returns a new BlockStore with the given DB,
// initialized to the last height that was committed to the DB.
func NewBlockStore(db dbm.DB) *BlockStore {
	bs := LoadBlockStoreState(db)
	return &BlockStore{
		lastCelestiaHeightQueried:   bs.LastCelestiaHeightQueried,
		base: bs.Base,
		height: bs.Height,
		db:     db,
	}
}

// Empty of contiguous block (may contain non-contiguous blocks)
func (bs *BlockStore) IsEmpty() bool {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.base == bs.height && bs.base == 0
}

// Base returns the first known contiguous block height, or 0 for empty block stores.
func (bs *BlockStore) Base() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.base
}

func (bs *BlockStore) LastCelestiaHeightQueried() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.lastCelestiaHeightQueried
}

// Height returns the last known contiguous block height, or 0 for empty block stores.
func (bs *BlockStore) Height() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.height
}

// LoadBlock returns the block with the given height.
// If no block is found for that height, it returns nil.
func (bs *BlockStore) LoadBlock(height int64) *cmtproto.Block {
	partsTotal := bs.LoadBlockMeta(height)
	if partsTotal == 0 {
		return nil
	}

	pbb := new(cmtproto.Block)
	buf := []byte{}
	for i := 0; i < int(partsTotal); i++ {
		part := bs.LoadBlockPart(height, i)
		// If the part is missing (e.g. since it has been deleted after we
		// loaded the block meta) we consider the whole block to be missing.
		if part == nil {
			return nil
		}
		buf = append(buf, part...)
	}
	err := proto.Unmarshal(buf, pbb)
	if err != nil {
		// NOTE: The existence of meta should imply the existence of the
		// block. So, make sure meta is only saved after blocks are saved.
		panic(fmt.Sprintf("Error reading block: %v", err))
	}

	return pbb
}

// LoadBlockPart returns the Part at the given index
// from the block at the given height.
// If no part is found for the given height and index, it returns nil.
func (bs *BlockStore) LoadBlockPart(height int64, index int) []byte {
	bz, err := bs.db.Get(calcBlockPartKey(height, index))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil
	}

	return bz
}

// LoadBlockMeta returns the BlockMeta for the given height.
// If no block is found for the given height, it returns nil.
func (bs *BlockStore) LoadBlockMeta(height int64) uint32 {
	bz, err := bs.db.Get(calcBlockMetaKey(height))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return 0
	}

	blockMeta := binary.LittleEndian.Uint32(bz)

	return blockMeta
}

// PruneBlocks removes block up to (but not including) a height. It returns number of blocks pruned and the evidence retain height - the height at which data needed to prove evidence must not be removed.
func (bs *BlockStore) PruneBlocks(height int64) (uint64, error) {
	if height <= 0 {
		return 0, fmt.Errorf("height must be greater than 0")
	}
	bs.mtx.RLock()
	if height > bs.height {
		bs.mtx.RUnlock()
		return 0, fmt.Errorf("cannot prune beyond the latest height %v", bs.height)
	}
	base := bs.base
	bs.mtx.RUnlock()
	if height < base {
		return 0, fmt.Errorf("cannot prune to height %v, it is lower than base height %v",
			height, base)
	}

	pruned := uint64(0)
	batch := bs.db.NewBatch()
	defer batch.Close()
	flush := func(batch dbm.Batch, base int64) error {
		// We can't trust batches to be atomic, so update base first to make sure noone
		// tries to access missing blocks.
		bs.mtx.Lock()
		defer batch.Close()
		defer bs.mtx.Unlock()
		bs.base = base
		return bs.saveStateAndWriteDB(batch, "failed to prune")
	}

	for h := base; h < height; h++ {
		totalParts := bs.LoadBlockMeta(h)
		if totalParts == 0 { // assume already deleted
			continue
		}

		if err := batch.Delete(calcBlockMetaKey(h)); err != nil {
			return 0, err
		}

		for p := 0; p < int(totalParts); p++ {
			if err := batch.Delete(calcBlockPartKey(h, p)); err != nil {
				return 0, err
			}
		}
		pruned++

		// flush every 1000 blocks to avoid batches becoming too large
		if pruned%1000 == 0 && pruned > 0 {
			err := flush(batch, h)
			if err != nil {
				return 0, err
			}
			batch = bs.db.NewBatch()
			defer batch.Close()
		}
	}

	err := flush(batch, height)
	if err != nil {
		return 0, err
	}
	return pruned, nil
}

// SaveBlock persists the given block, blockParts, and seenCommit to the underlying db.
// blockParts: Must be parts of the block
// seenCommit: The +2/3 precommits that were seen which committed at height.
//
//	If all the nodes restart after committing a block,
//	we need this to reload the precommits to catch-up nodes to the
//	most recent height.  Otherwise they'd stall at H-1.
func (bs *BlockStore) SaveBlock(celestiaHeight int64, height int64, block []byte) {
	if block == nil {
		panic("BlockStore can only save a non-nil block")
	}

	batch := bs.db.NewBatch()
	defer batch.Close()

	if err := bs.saveBlockToBatch(height, block, batch); err != nil {
		panic(err)
	}

	bs.mtx.Lock()
	defer bs.mtx.Unlock()

	if bs.base == 0 {
		bs.base = height
	}
	
	if bs.height + 1 == height {
		bs.height = height
		nextHeight := height + 1
		hasNextBlock, _ := bs.db.Has(calcBlockMetaKey(nextHeight))
		for ; hasNextBlock ; {
			bs.height = nextHeight
			nextHeight++
			hasNextBlock, _ = bs.db.Has(calcBlockMetaKey(nextHeight)) 
		}
	}

	if celestiaHeight > bs.lastCelestiaHeightQueried {
		bs.lastCelestiaHeightQueried = celestiaHeight
	}

	// Save new BlockStoreState descriptor. This also flushes the database.
	err := bs.saveStateAndWriteDB(batch, "failed to save block")
	if err != nil {
		panic(err)
	}
}

func (bs *BlockStore) saveBlockToBatch(
	height int64,
	block []byte,
	batch dbm.Batch) error {

	if block == nil {
		panic("BlockStore can only save a non-nil block")
	}

	partSet := NewPartSetFromData(block, BlockPartSizeBytes)

	// If the block is small, batch save the block parts. Otherwise, save the
	// parts individually.
	saveBlockPartsToBatch := partSet.Total() <= maxBlockPartsToBatch

	// Save block parts. This must be done before the block meta, since callers
	// typically load the block meta first as an indication that the block exists
	// and then go on to load block parts - we must make sure the block is
	// complete as soon as the block meta is written.
	for i := 0; i < int(partSet.Total()); i++ {
		part := partSet.GetPart(i)
		bs.saveBlockPart(height, i, part, batch, saveBlockPartsToBatch)
	}

	// Save block meta
	blockMeta := make([]byte, 4)
	binary.LittleEndian.PutUint32(blockMeta, partSet.total)
	if err := batch.Set(calcBlockMetaKey(height), blockMeta); err != nil {
		return err
	}

	return nil
}

func (bs *BlockStore) saveBlockPart(height int64, index int, part []byte, batch dbm.Batch, saveBlockPartsToBatch bool) {
	var err error
	if saveBlockPartsToBatch {
		err = batch.Set(calcBlockPartKey(height, index), part)
	} else {
		err = bs.db.Set(calcBlockPartKey(height, index), part)
	}
	if err != nil {
		panic(err)
	}
}

// Contract: the caller MUST have, at least, a read lock on `bs`.
func (bs *BlockStore) saveStateAndWriteDB(batch dbm.Batch, errMsg string) error {
	bss := BlockStoreState{
		Base: bs.base,
		Height: bs.height,
		LastCelestiaHeightQueried: bs.lastCelestiaHeightQueried,
	}
	SaveBlockStoreStateBatch(&bss, batch)

	err := batch.WriteSync()
	if err != nil {
		return fmt.Errorf("error writing batch to DB %q: (lastCelestiaHeightQueried %d, height %d): %w",
			errMsg, bs.lastCelestiaHeightQueried, bs.height, err)
	}
	return nil
}

func (bs *BlockStore) Close() error {
	return bs.db.Close()
}

//-----------------------------------------------------------------------------

func calcBlockMetaKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func calcBlockPartKey(height int64, partIndex int) []byte {
	return []byte(fmt.Sprintf("P:%v:%v", height, partIndex))
}

//-----------------------------------------------------------------------------

var blockStoreKey = []byte("blockStore")

// SaveBlockStoreStateBatch persists the blockStore state to the database.
// It uses the DB batch passed as parameter
func SaveBlockStoreStateBatch(bsj *BlockStoreState, batch dbm.Batch) {
	saveBlockStoreStateBatchInternal(bsj, nil, batch)
}

func saveBlockStoreStateBatchInternal(bsj *BlockStoreState, db dbm.DB, batch dbm.Batch) {
	bytes, err := json.Marshal(bsj)
	if err != nil {
		panic(fmt.Sprintf("could not marshal state bytes: %v", err))
	}
	if batch != nil {
		err = batch.Set(blockStoreKey, bytes)
	} else {
		if db == nil {
			panic("both 'db' and 'batch' cannot be nil")
		}
		err = db.SetSync(blockStoreKey, bytes)
	}
	if err != nil {
		panic(err)
	}
}

// LoadBlockStoreState returns the BlockStoreState as loaded from disk.
// If no BlockStoreState was previously persisted, it returns the zero value.
func LoadBlockStoreState(db dbm.DB) BlockStoreState {
	bytes, err := db.Get(blockStoreKey)
	if err != nil {
		panic(err)
	}

	if len(bytes) == 0 {
		return BlockStoreState{
			Base: 0,
			Height: 0,
			LastCelestiaHeightQueried:   0,
		}
	}

	var bsj BlockStoreState
	if err := json.Unmarshal(bytes, &bsj); err != nil {
		panic(fmt.Sprintf("Could not unmarshal bytes: %X", bytes))
	}

	return bsj
}
