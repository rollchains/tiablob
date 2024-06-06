package consensus

import (
	cstypes "github.com/cometbft/cometbft/consensus/types"
	//cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/p2p"
	cmtcons "github.com/cometbft/cometbft/proto/tendermint/consensus"
	sm "github.com/cometbft/cometbft/state"
	//"github.com/cometbft/cometbft/types"
)

const (
	StateChannel       = byte(0x20)
	DataChannel        = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)

	maxMsgSize = 1048576 // 1MB; NOTE/TODO: keep in sync with types.PartSet sizes.
)

// Reactor defines a reactor for the consensus service.
type Reactor struct {
	p2p.BaseReactor // BaseService + p2p.Switch

	localPeerID p2p.ID
}

// NewReactor returns a new Reactor 
func NewReactor(localPeerID p2p.ID) *Reactor {
	conR := &Reactor{
		localPeerID: localPeerID,
	}
	conR.BaseReactor = *p2p.NewBaseReactor("Consensus", conR)

	return conR
}

// OnStart implements BaseService by subscribing to events, which later will be
// broadcasted to other peers and starting state if we're not in block sync.
func (conR *Reactor) OnStart() error {
 	return nil
}

// OnStop implements BaseService by unsubscribing from events and stopping
// state.
func (conR *Reactor) OnStop() {

}

// SwitchToConsensus switches from block_sync mode to consensus mode.
// It resets the state, turns off block_sync, and starts the consensus state-machine
func (conR *Reactor) SwitchToConsensus(state sm.State, skipWAL bool) {

}

// GetChannels implements Reactor
func (conR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
	// TODO optimize
	return []*p2p.ChannelDescriptor{
		{
			ID:                  StateChannel,
			Priority:            6,
			SendQueueCapacity:   100,
			RecvMessageCapacity: maxMsgSize,
			MessageType:         &cmtcons.Message{},
		},
	}
}

// // InitPeer implements Reactor by creating a state for the peer.
// func (conR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
// 	//peerState := NewPeerState(peer).SetLogger(conR.Logger)
// 	//peer.Set(types.PeerStateKey, peerState)
// 	return peer
// }

// AddPeer implements Reactor by spawning multiple gossiping goroutines for the
// peer.
func (conR *Reactor) AddPeer(peer p2p.Peer) {
	if peer.ID() == conR.localPeerID {
		conR.sendNewRoundStepMessage(peer)
	}
}

// // RemovePeer is a noop.
// func (conR *Reactor) RemovePeer(p2p.Peer, interface{}) {
// }

// Receive implements Reactor
// NOTE: We process these messages even when we're block_syncing.
// Messages affect either a peer state or the consensus state.
// Peer state updates can happen in parallel, but processing of
// proposals, block parts, and votes are ordered by the receiveRoutine
// NOTE: blocks on consensus state for proposals, block parts, and votes
// func (conR *Reactor) Receive(e p2p.Envelope) {

// }

// SetEventBus sets event bus.
// func (conR *Reactor) SetEventBus(b *types.EventBus) {
// }

// WaitSync returns whether the consensus reactor is waiting for state/block sync.
// func (conR *Reactor) WaitSync() bool {
// 	return true
// }

func makeRoundStepMessage() (nrsMsg *cmtcons.NewRoundStep) {
	nrsMsg = &cmtcons.NewRoundStep{
		Height:                1000,
		Round:                 0,
		Step:                  1,
		SecondsSinceStartTime: 1,
		LastCommitRound:       0,
	}
	return
}

func (conR *Reactor) sendNewRoundStepMessage(peer p2p.Peer) {
	nrsMsg := makeRoundStepMessage()
	peer.Send(p2p.Envelope{
		ChannelID: StateChannel,
		Message:   nrsMsg,
	})
}

// String returns a string representation of the Reactor.
// NOTE: For now, it is just a hard-coded string to avoid accessing unprotected shared variables.
// TODO: improve!
func (conR *Reactor) String() string {
	// better not to access shared variables
	return "ConsensusReactor" // conR.StringIndented("")
}

// NewRoundStepMessage is sent for every step taken in the ConsensusState.
// For every height/round/step transition
type NewRoundStepMessage struct {
	Height                int64
	Round                 int32
	Step                  cstypes.RoundStepType
	SecondsSinceStartTime int64
	LastCommitRound       int32
}
