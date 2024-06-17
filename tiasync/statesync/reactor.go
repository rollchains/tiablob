package statesync

import (
	"github.com/cometbft/cometbft/config"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/p2p"
	ssproto "github.com/cometbft/cometbft/proto/tendermint/statesync"
)

const (
	// SnapshotChannel exchanges snapshot metadata
	SnapshotChannel = byte(0x60)
	// ChunkChannel exchanges chunk contents
	ChunkChannel = byte(0x61)
)

// Reactor handles state sync, both restoring snapshots for the local node and serving snapshots
// for other nodes.
type Reactor struct {
	p2p.BaseReactor

	cfg       config.StateSyncConfig
	localPeerID p2p.ID
	localPeer p2p.Peer
	remotePeer p2p.Peer

	// This will only be set when a state sync is in progress. It is used to feed received
	// snapshots and chunks into the sync.
	mtx    cmtsync.RWMutex
}

// NewReactor creates a new state sync reactor.
// Our state sync reactor is simple. It will only forward req/resp to and from our first remote peer.
// Our first remote peer is expected to be one of our upstream/persistent peers.
// If that remote peer results in our local node/peer blacklisting us, i.e. rejects the remote peer's snapshot/chunks,
// user will need to remove that upstream peer and restart the node.
// Improvements can be made to query all our remote peers, choose the best snapshot, and serve that snapshot to
// our local node. Snapshot discovery time may need to increase since the local node's state sync may have started
// before tiasync has connected to it. For example, a 15 second discovery time could be less due to this latency.
// If we query multiple nodes for the best snapshot, local node/cometbft's discovery time will need to be:
// tiasync startup latency + tiasync discovery time + tiasync snapshot response latency.
func NewReactor(
	cfg config.StateSyncConfig,
	localPeerID p2p.ID,
) *Reactor {
	r := &Reactor{
		cfg:       cfg,
		localPeerID: localPeerID,
	}
	r.BaseReactor = *p2p.NewBaseReactor("StateSync", r)

	return r
}

// GetChannels implements p2p.Reactor.
func (r *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  SnapshotChannel,
			Priority:            5,
			SendQueueCapacity:   10,
			RecvMessageCapacity: snapshotMsgSize,
			MessageType:         &ssproto.Message{},
		},
		{
			ID:                  ChunkChannel,
			Priority:            3,
			SendQueueCapacity:   10,
			RecvMessageCapacity: chunkMsgSize,
			MessageType:         &ssproto.Message{},
		},
	}
}

// OnStart implements p2p.Reactor.
func (r *Reactor) OnStart() error {
	return nil
}

// AddPeer implements p2p.Reactor.
func (r *Reactor) AddPeer(peer p2p.Peer) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.localPeerID == peer.ID() {
		r.localPeer = peer
	} else if r.remotePeer == nil {
		r.Logger.Info("Remote peer added", "peer id", peer.ID())
		r.remotePeer = peer
	}
}

// RemovePeer implements p2p.Reactor.
func (r *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.localPeerID == peer.ID() {
		r.localPeer = nil
	} else if r.remotePeer != nil && r.remotePeer.ID() == peer.ID() {
		r.remotePeer = nil
	}
}

// Receive implements p2p.Reactor.
// Forwards requests to our first remote peer (upstream/persistent peer)
// Forwards responses to our local node
func (r *Reactor) Receive(e p2p.Envelope) {
	if !r.IsRunning() {
		return
	}

	err := validateMsg(e.Message)
	if err != nil {
		r.Logger.Error("Invalid message", "peer", e.Src, "msg", e.Message, "err", err)
		r.Switch.StopPeerForError(e.Src, err)
		return
	}

	switch e.ChannelID {
	case SnapshotChannel:
		switch msg := e.Message.(type) {
		case *ssproto.SnapshotsRequest:
			if r.remotePeer != nil {
				r.remotePeer.Send(p2p.Envelope{
					ChannelID: SnapshotChannel,
					Message:   &ssproto.SnapshotsRequest{},
				})
			}
		case *ssproto.SnapshotsResponse:
			if r.localPeer != nil {
				r.localPeer.Send(p2p.Envelope{
					ChannelID: e.ChannelID,
					Message: msg,
				})
			}

		default:
			r.Logger.Error("Received unknown message %T", msg)
		}

	case ChunkChannel:
		switch msg := e.Message.(type) {
		case *ssproto.ChunkRequest:
			if r.remotePeer != nil {
				r.remotePeer.Send(p2p.Envelope{
					ChannelID: ChunkChannel,
					Message:   msg,
				})
			}
		case *ssproto.ChunkResponse:
			if r.localPeer != nil {
				r.localPeer.Send(p2p.Envelope{
					ChannelID: e.ChannelID,
					Message: msg,
				})
			}

		default:
			r.Logger.Error("Received unknown message %T", msg)
		}

	default:
		r.Logger.Error("Received message on invalid channel %x", e.ChannelID)
	}
}