package blocksync

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	bcproto "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	//sm "github.com/cometbft/cometbft/state"
	//"github.com/cometbft/cometbft/store"
	"github.com/cometbft/cometbft/types"

	"github.com/rollchains/tiablob/tiasync/store"
	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider"
)

const (
	// BlocksyncChannel is a channel for blocks and status updates (`BlockStore` height)
	BlocksyncChannel = byte(0x40)
)

// Reactor handles long-term catchup syncing.
type Reactor struct {
	p2p.BaseReactor

	store         *store.BlockStore

	// Have we started querying celestia?
	localPeerInBlockSync bool

	localPeerID p2p.ID

	blockProvider *blockprovider.BlockProvider

	metrics *Metrics
}

// NewReactor returns new reactor instance.
func NewReactor(store *store.BlockStore, localPeerID p2p.ID,
	metrics *Metrics, celestiaCfg *relayer.CelestiaConfig, genTime time.Time,
) *Reactor {
	// Get the last height queried and if available, redo that query to ensure we got everything
	celestiaHeight := store.LastCelestiaHeightQueried()-1

	bcR := &Reactor{
		localPeerID:  localPeerID,
		store:        store,
		localPeerInBlockSync: false,
		metrics:      metrics,
		blockProvider:    blockprovider.NewBlockProvider(store, celestiaHeight, celestiaCfg, genTime),
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("Reactor", bcR)

	return bcR
}

// SetLogger implements service.Service by setting the logger on reactor and pool.
func (bcR *Reactor) SetLogger(l log.Logger) {
	bcR.BaseService.Logger = l
	bcR.blockProvider.SetLogger(l)
}

// OnStart implements service.Service.
func (bcR *Reactor) OnStart() error {
	return nil
}

// OnStop implements service.Service.
func (bcR *Reactor) OnStop() {}

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
	if peer.ID() == bcR.localPeerID {
		peer.Send(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message: &bcproto.StatusResponse{
				Base:  int64(0), 
				Height: math.MaxInt64-1, // Send max to keep int block sync mode, peer won't request >20 blocks at the same time
			},
		})
	}	
	// it's OK if send fails. will try later in poolRoutine

	// peer is added to the pool once we receive the first
	// bcStatusResponseMessage from the peer and call pool.SetPeerRange
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {}

// respondToPeer loads a block and sends it to the requesting peer,
// if we have it. Otherwise, we'll respond saying we don't have it.
func (bcR *Reactor) respondToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	//block := bcR.store.LoadBlock(msg.Height)
	block := bcR.blockProvider.GetBlock(msg.Height)
	if block == nil {
		bcR.Logger.Info("Peer asking for a block we don't have", "src", src, "height", msg.Height)
		return src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.NoBlockResponse{Height: msg.Height},
		})
	}

	// Ask local peer for their height for pruning (every 10 blocks sent)
	if msg.Height % 10 == 0 {
		src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.StatusRequest{},
		})
	}

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
		if e.Src.ID() == bcR.localPeerID {
			if !bcR.localPeerInBlockSync{
				// At this point, we start querying celestia. We don't start when the reactor is created because of state sync.
				// We know our local node has entered block sync,
				// and if we state sync'd, we will have our celestia da light client state with a latest height to start querying from.
				bcR.localPeerInBlockSync = true
				go bcR.blockProvider.Start()
			}
			bcR.respondToPeer(msg, e.Src)
		}
	case *bcproto.StatusRequest:
		bcR.Logger.Debug("block sync Receive StatusRequest")
		// Send peer our state.
		if e.Src.ID() == bcR.localPeerID {
			e.Src.TrySend(p2p.Envelope{
				ChannelID: BlocksyncChannel,
				Message: &bcproto.StatusResponse{
					Height: math.MaxInt64-1, // Send max to keep int block sync mode, peer won't request >20 blocks at the same time
					Base:   int64(0),
				},
			})
		}
	case *bcproto.StatusResponse:
		bcR.Logger.Debug("block sync Receive StatusResponse", "height", msg.Height)
		if e.Src.ID() == bcR.localPeerID {
			pruned, _ := bcR.store.PruneBlocks(msg.Height)
			bcR.Logger.Debug("blocks pruned", "pruned", pruned)
		}
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
