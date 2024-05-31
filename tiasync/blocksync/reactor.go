package blocksync

import (
	"fmt"
	"reflect"
	//"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	bcproto "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	sm "github.com/cometbft/cometbft/state"
	//"github.com/cometbft/cometbft/store"
	"github.com/cometbft/cometbft/types"

	"github.com/rollchains/tiablob/tiasync/store"
	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/tiasync/celestia"
)

const (
	// BlocksyncChannel is a channel for blocks and status updates (`BlockStore` height)
	BlocksyncChannel = byte(0x40)

	trySyncIntervalMS = 10

	// stop syncing when last block's time is
	// within this much of the system time.
	// stopSyncingDurationMinutes = 10

	// ask for best height every 10s
	statusUpdateIntervalSeconds = 10
	// check if we should switch to consensus reactor
	switchToConsensusIntervalSeconds = 1
)

type consensusReactor interface {
	// for when we switch from blocksync reactor and block sync to
	// the consensus machine
	SwitchToConsensus(state sm.State, skipWAL bool)
}

type peerError struct {
	err    error
	peerID p2p.ID
}

func (e peerError) Error() string {
	return fmt.Sprintf("error with peer %v: %s", e.peerID, e.err.Error())
}

// Reactor handles long-term catchup syncing.
type Reactor struct {
	p2p.BaseReactor

	// immutable
	initialState sm.State

	//blockExec     *sm.BlockExecutor
	store         sm.BlockStore
	blockSync     bool

	blockPool *celestia.BlockPool

	metrics *Metrics
}

// NewReactor returns new reactor instance.
func NewReactor(state sm.State, store *store.BlockStore,
	blockSync bool, metrics *Metrics, tiasyncCfg *relayer.CelestiaConfig, logger log.Logger,
) *Reactor {

	// storeHeight := store.Height()
	// if storeHeight == 0 {
	// 	// If state sync was performed offline and the stores were bootstrapped to height H
	// 	// the state store's lastHeight will be H while blockstore's Height and Base are still 0
	// 	// 1. This scenario should not lead to a panic in this case, which is indicated by
	// 	// having a OfflineStateSyncHeight > 0
	// 	// 2. We need to instruct the blocksync reactor to start fetching blocks from H+1
	// 	// instead of 0.
	// 	storeHeight = offlineStateSyncHeight
	// }
	// if state.LastBlockHeight != storeHeight {
	// 	panic(fmt.Sprintf("state (%v) and store (%v) height mismatch, stores were left in an inconsistent state", state.LastBlockHeight,
	// 		storeHeight))
	// }

	// // It's okay to block since sendRequest is called from a separate goroutine
	// // (bpRequester#requestRoutine; 1 per each peer).
	// //requestsCh := make(chan BlockRequest)

	// //const capacity = 1000                      // must be bigger than peers count
	// //errorsCh := make(chan peerError, capacity) // so we don't block in #Receive#pool.AddBlock

	// startHeight := storeHeight + 1
	// if startHeight == 1 {
	// 	startHeight = state.InitialHeight
	// }

	bcR := &Reactor{
		initialState: state,
		//blockExec:    blockExec,
		store:        store,
		blockSync:    blockSync,
		metrics:      metrics,
		blockPool:    celestia.NewBlockPool(0, tiasyncCfg, logger.With("tsmodule", "tsblockpool")),
	}
	bcR.SetLogger(logger.With("tsmodule", "tsblocksync"))
	go bcR.blockPool.Start()
	bcR.BaseReactor = *p2p.NewBaseReactor("Reactor", bcR)
	return bcR
}

// SetLogger implements service.Service by setting the logger on reactor and pool.
func (bcR *Reactor) SetLogger(l log.Logger) {
	bcR.BaseService.Logger = l
	//bcR.pool.Logger = l
}

// OnStart implements service.Service.
func (bcR *Reactor) OnStart() error {
	bcR.Logger.Debug("block sync OnStart()")
	// TODO: start get blocks from celestia
	if bcR.blockSync {
		// err := bcR.pool.Start()
		// if err != nil {
		// 	return err
		// }
		// bcR.poolRoutineWg.Add(1)
		// go func() {
		// 	defer bcR.poolRoutineWg.Done()
		// 	bcR.poolRoutine(false)
		// }()
	}
	return nil
}

// SwitchToBlockSync is called by the state sync reactor when switching to block sync.
func (bcR *Reactor) SwitchToBlockSync(state sm.State) error {
	bcR.Logger.Debug("block sync SwitchToBlockSync()")
	bcR.blockSync = true
	bcR.initialState = state

	//go bcR.blockPool.Start()
	// TODO: start getting blocks from celestia

	// bcR.pool.height = state.LastBlockHeight + 1
	// err := bcR.pool.Start()
	// if err != nil {
	// 	return err
	// }
	// bcR.poolRoutineWg.Add(1)
	// go func() {
	// 	defer bcR.poolRoutineWg.Done()
	// 	bcR.poolRoutine(true)
	// }()
	return nil
}

// OnStop implements service.Service.
func (bcR *Reactor) OnStop() {
	bcR.Logger.Debug("block sync OnStop()")
	if bcR.blockSync {
		// TODO: stop getting blocks from celestia

		// if err := bcR.pool.Stop(); err != nil {
		// 	bcR.Logger.Error("Error stopping pool", "err", err)
		// }
		// bcR.poolRoutineWg.Wait()
	}
}

// GetChannels implements Reactor
func (bcR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  BlocksyncChannel,
			Priority:            5,
			SendQueueCapacity:   1000,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: MaxMsgSize,
			MessageType:         &bcproto.Message{},
		},
	}
}

// AddPeer implements Reactor by sending our state to peer.
func (bcR *Reactor) AddPeer(peer p2p.Peer) {
	peer.Send(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message: &bcproto.StatusResponse{
			Base:  int64(0), 
			//Base:   bcR.store.Base(),
			Height: bcR.blockPool.GetHeight()+5,
			//Height: bcR.store.Height(),
		},
	})
	// it's OK if send fails. will try later in poolRoutine

	// peer is added to the pool once we receive the first
	// bcStatusResponseMessage from the peer and call pool.SetPeerRange
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {
	//bcR.pool.RemovePeer(peer.ID())
}

// respondToPeer loads a block and sends it to the requesting peer,
// if we have it. Otherwise, we'll respond saying we don't have it.
func (bcR *Reactor) respondToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	//block := bcR.store.LoadBlock(msg.Height)
	block := bcR.blockPool.GetBlock(msg.Height)
	if block == nil {
		bcR.Logger.Info("Peer asking for a block we don't have", "src", src, "height", msg.Height)
		return src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.NoBlockResponse{Height: msg.Height},
		})
	}

	//bl, err := block.ToProto()
	//if err != nil {
	//	bcR.Logger.Error("could not convert msg to protobuf", "err", err)
	//	return false
	//}

	var extCommit *types.ExtendedCommit
	return src.TrySend(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message: &bcproto.BlockResponse{
			Block:     block,
			ExtCommit: extCommit.ToProto(),
		},
	})
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (bcR *Reactor) Receive(e p2p.Envelope) {
	if err := ValidateMsg(e.Message); err != nil {
		bcR.Logger.Error("Peer sent us invalid msg", "peer", e.Src, "msg", e.Message, "err", err)
		bcR.Switch.StopPeerForError(e.Src, err)
		return
	}

	bcR.Logger.Debug("Receive", "e.Src", e.Src, "chID", e.ChannelID, "msg", e.Message)

	switch msg := e.Message.(type) {
	case *bcproto.BlockRequest:
		bcR.Logger.Debug("block sync Receive BlockRequest", "height", msg.Height)
		bcR.respondToPeer(msg, e.Src)
	case *bcproto.StatusRequest:
		bcR.Logger.Debug("block sync Receive StatusRequest")
		// Send peer our state.
		e.Src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message: &bcproto.StatusResponse{
				Height: bcR.blockPool.GetHeight()+5,
				//Height: bcR.store.Height(),
				Base:   int64(0),
				//Base:   bcR.store.Base(),
			},
		})
	case *bcproto.StatusResponse:
		bcR.Logger.Debug("block sync Receive StatusResponse", "height", msg.Height)
		// TODO: prune block store
		//_, _, _ := bcR.store.PruneBlocks(msg.Height, bcR.state??)
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

// Handle messages from the poolReactor telling the reactor what to do.
// NOTE: Don't sleep in the FOR_LOOP or otherwise slow it down!
/*func (bcR *Reactor) poolRoutine(stateSynced bool) {
	bcR.metrics.Syncing.Set(1)
	defer bcR.metrics.Syncing.Set(0)

	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
	defer trySyncTicker.Stop()

	pruneTicker := time.NewTicker(10 * time.Second)
	defer trySyncTicker.Stop()

	blocksSynced := uint64(0)

	chainID := bcR.initialState.ChainID
	state := bcR.initialState

	lastHundred := time.Now()
	lastRate := 0.0

	didProcessCh := make(chan struct{}, 1)

FOR_LOOP:
	for {
		select {
		case <-pruneTicker.C:
			go bcR.BroadcastStatusRequest()

		case <-trySyncTicker.C: // chan time
			select {
			case didProcessCh <- struct{}{}:
			default:
			}

		case <-didProcessCh:
			// NOTE: It is a subtle mistake to process more than a single block
			// at a time (e.g. 10) here, because we only TrySend 1 request per
			// loop.  The ratio mismatch can result in starving of blocks, a
			// sudden burst of requests and responses, and repeat.
			// Consequently, it is better to split these routines rather than
			// coupling them as it's written here.  TODO uncouple from request
			// routine.

			// See if there are any blocks to sync.
			first, second, extCommit := bcR.pool.PeekTwoBlocks()
			if first == nil || second == nil {
				// we need to have fetched two consecutive blocks in order to
				// perform blocksync verification
				continue FOR_LOOP
			}
			// Some sanity checks on heights
			if state.LastBlockHeight > 0 && state.LastBlockHeight+1 != first.Height {
				// Panicking because the block pool's height  MUST keep consistent with the state; the block pool is totally under our control
				panic(fmt.Errorf("peeked first block has unexpected height; expected %d, got %d", state.LastBlockHeight+1, first.Height))
			}
			if first.Height+1 != second.Height {
				// Panicking because this is an obvious bug in the block pool, which is totally under our control
				panic(fmt.Errorf("heights of first and second block are not consecutive; expected %d, got %d", state.LastBlockHeight, first.Height))
			}
	
			// Before priming didProcessCh for another check on the next
			// iteration, break the loop if the BlockPool or the Reactor itself
			// has quit. This avoids case ambiguity of the outer select when two
			// channels are ready.
			if !bcR.IsRunning() { //|| !bcR.pool.IsRunning() {
				break FOR_LOOP
			}
			// Try again quickly next loop.
			didProcessCh <- struct{}{}

			firstParts, err := first.MakePartSet(types.BlockPartSizeBytes)
			if err != nil {
				bcR.Logger.Error("failed to make ",
					"height", first.Height,
					"err", err.Error())
				break FOR_LOOP
			}
			firstPartSetHeader := firstParts.Header()
			firstID := types.BlockID{Hash: first.Hash(), PartSetHeader: firstPartSetHeader}
			// Finally, verify the first block using the second's commit
			// NOTE: we can probably make this more efficient, but note that calling
			// first.Hash() doesn't verify the tx contents, so MakePartSet() is
			// currently necessary.
			// TODO(sergio): Should we also validate against the extended commit?
			err = state.Validators.VerifyCommitLight(
				chainID, firstID, first.Height, second.LastCommit)

			// if err == nil {
			// 	// validate the block before we persist it
			// 	err = bcR.blockExec.ValidateBlock(state, first)
			// }
			
			// We use LastCommit here instead of extCommit. extCommit is not
			// guaranteed to be populated by the peer if extensions are not enabled.
			// Currently, the peer should provide an extCommit even if the vote extension data are absent
			// but this may change so using second.LastCommit is safer.
			bcR.store.SaveBlock(first, firstParts, second.LastCommit)

			// TODO: same thing for app - but we would need a way to
			// get the hash without persisting the state
			// state, err = bcR.blockExec.ApplyBlock(state, firstID, first)
			// if err != nil {
			// 	// TODO This is bad, are we zombie?
			// 	panic(fmt.Sprintf("Failed to process committed block (%d:%X): %v", first.Height, first.Hash(), err))
			// }
			bcR.metrics.recordBlockMetrics(first)
			blocksSynced++

			if blocksSynced%100 == 0 {
				lastRate = 0.9*lastRate + 0.1*(100/time.Since(lastHundred).Seconds())
				bcR.Logger.Info("Block Sync Rate", //"height", bcR.pool.height,
					//"max_peer_height", bcR.pool.MaxPeerHeight(), 
					"blocks/s", lastRate)
				lastHundred = time.Now()
			}

			continue FOR_LOOP

		case <-bcR.Quit():
			break FOR_LOOP
		}
	}
}*/

// BroadcastStatusRequest broadcasts `BlockStore` base and height.
func (bcR *Reactor) BroadcastStatusRequest() {
	bcR.Switch.Broadcast(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.StatusRequest{},
	})
}