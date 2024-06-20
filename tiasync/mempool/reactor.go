package mempool

import (
	"context"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	protomem "github.com/cometbft/cometbft/proto/tendermint/mempool"
	"golang.org/x/sync/semaphore"
)

const (
	MempoolChannel = byte(0x30)
)

// Reactor handles mempool tx broadcasting amongst peers.
// It maintains a map from peer ID to peer. Broadcasting is only outward to val network.
type Reactor struct {
	p2p.BaseReactor
	config      *cfg.MempoolConfig
	peers       map[p2p.ID]p2p.Peer
	localPeerID p2p.ID

	// Semaphores to keep track of how many connections to peers are active for broadcasting
	// transactions. Each semaphore has a capacity that puts an upper bound on the number of
	// connections for different groups of peers.
	activePersistentPeersSemaphore    *semaphore.Weighted
	activeNonPersistentPeersSemaphore *semaphore.Weighted
}

// NewReactor returns a new Reactor with the given config.
func NewReactor(config *cfg.MempoolConfig, localPeerID p2p.ID) *Reactor {
	memR := &Reactor{
		config:      config,
		peers:       make(map[p2p.ID]p2p.Peer),
		localPeerID: localPeerID,
	}
	memR.BaseReactor = *p2p.NewBaseReactor("Mempool", memR)
	memR.activePersistentPeersSemaphore = semaphore.NewWeighted(int64(memR.config.ExperimentalMaxGossipConnectionsToPersistentPeers))
	memR.activeNonPersistentPeersSemaphore = semaphore.NewWeighted(int64(memR.config.ExperimentalMaxGossipConnectionsToNonPersistentPeers))

	return memR
}

// OnStart implements p2p.BaseReactor.
func (memR *Reactor) OnStart() error {
	if !memR.config.Broadcast {
		memR.Logger.Info("Tx broadcasting is disabled")
	}
	return nil
}

// GetChannels implements Reactor by returning the list of channels for this
// reactor.
func (memR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	largestTx := make([]byte, memR.config.MaxTxBytes)
	batchMsg := protomem.Message{
		Sum: &protomem.Message_Txs{
			Txs: &protomem.Txs{Txs: [][]byte{largestTx}},
		},
	}

	return []*p2p.ChannelDescriptor{
		{
			ID:                  MempoolChannel,
			Priority:            5,
			RecvMessageCapacity: batchMsg.Size(),
			MessageType:         &protomem.Message{},
		},
	}
}

// AddPeer implements Reactor.
// It starts a broadcast routine ensuring all txs are forwarded to the given peer.
func (memR *Reactor) AddPeer(peer p2p.Peer) {
	memR.Logger.Debug("AddPeer", "peer", peer.ID())
	if memR.config.Broadcast {
		if memR.localPeerID != peer.ID() {
			memR.peers[peer.ID()] = peer
		}
	}
}

// RemovePeer implements Reactor.
func (memR *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {
	memR.Logger.Debug("RemovePeer", "peer", peer.ID())
	delete(memR.peers, peer.ID())
	// broadcast routine checks if peer is gone and returns
}

// Receive implements Reactor.
// It adds any received transactions to the mempool.
func (memR *Reactor) Receive(e p2p.Envelope) {
	memR.Logger.Debug("Receive", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
	switch msg := e.Message.(type) {
	case *protomem.Txs:
		protoTxs := msg.GetTxs()
		if len(protoTxs) == 0 {
			memR.Logger.Error("received empty txs from peer", "src", e.Src)
			return
		}

		// only broadcast outwards, drop any incoming external txs
		if e.Src.ID() == memR.localPeerID {
			memR.broadcastTxRoutine(protoTxs)
		}

	default:
		// note: tiasync only supports 1 message, so we may enter here occasionally
		memR.Logger.Debug("unknown message type", "src", e.Src, "chId", e.ChannelID, "msg", e.Message)
		return
	}

	// broadcasting happens from go routines per peer
}

func (memR *Reactor) broadcastTxRoutine(txs [][]byte) {
	for _, peer := range memR.peers {
		go memR.sendTxToPeer(peer, txs)
	}

}

func (memR *Reactor) sendTxToPeer(peer p2p.Peer, txs [][]byte) {
	// Always forward transactions to unconditional peers.
	if !memR.Switch.IsPeerUnconditional(peer.ID()) {
		// Depending on the type of peer, we choose a semaphore to limit the gossiping peers.
		var peerSemaphore *semaphore.Weighted
		if peer.IsPersistent() && memR.config.ExperimentalMaxGossipConnectionsToPersistentPeers > 0 {
			peerSemaphore = memR.activePersistentPeersSemaphore
		} else if !peer.IsPersistent() && memR.config.ExperimentalMaxGossipConnectionsToNonPersistentPeers > 0 {
			peerSemaphore = memR.activeNonPersistentPeersSemaphore
		}

		if peerSemaphore != nil {
			for peer.IsRunning() {
				// Block on the semaphore until a slot is available to start gossiping with this peer.
				// Do not block indefinitely, in case the peer is disconnected before gossiping starts.
				ctxTimeout, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
				// Block sending transactions to peer until one of the connections become
				// available in the semaphore.
				err := peerSemaphore.Acquire(ctxTimeout, 1)
				cancel()

				if err != nil {
					return
				}

				// Release semaphore to allow other peer to start sending transactions.
				defer peerSemaphore.Release(1)
				break
			}
		}
	}

	if !memR.IsRunning() || !peer.IsRunning() {
		return
	}

	memR.Logger.Debug("Sending txs", "peer", peer.ID())

	peer.Send(p2p.Envelope{
		ChannelID: MempoolChannel,
		Message:   &protomem.Txs{Txs: txs},
	})
}
