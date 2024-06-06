package consensus

import (
	"errors"
	"fmt"
	//"sync"
	//"time"

	cstypes "github.com/cometbft/cometbft/consensus/types"
	//"github.com/cometbft/cometbft/libs/bits"
	//cmtevents "github.com/cometbft/cometbft/libs/events"
	//cmtjson "github.com/cometbft/cometbft/libs/json"
	//"github.com/cometbft/cometbft/libs/log"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/p2p"
	cmtcons "github.com/cometbft/cometbft/proto/tendermint/consensus"
	//cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
	//cmttime "github.com/cometbft/cometbft/types/time"
)

const (
	StateChannel       = byte(0x20)
	DataChannel        = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)

	maxMsgSize = 1048576 // 1MB; NOTE/TODO: keep in sync with types.PartSet sizes.

	blocksToContributeToBecomeGoodPeer = 10000
	votesToContributeToBecomeGoodPeer  = 10000
)

//-----------------------------------------------------------------------------

// Reactor defines a reactor for the consensus service.
type Reactor struct {
	p2p.BaseReactor // BaseService + p2p.Switch

	mtx      cmtsync.RWMutex
	localPeerID p2p.ID
}

type ReactorOption func(*Reactor)

// NewReactor returns a new Reactor with the given
// consensusState.
func NewReactor(localPeerID p2p.ID) *Reactor {
	conR := &Reactor{
		//rs:       consensusState.GetRoundState(),
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

// InitPeer implements Reactor by creating a state for the peer.
func (conR *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	//peerState := NewPeerState(peer).SetLogger(conR.Logger)
	//peer.Set(types.PeerStateKey, peerState)
	return peer
}

// AddPeer implements Reactor by spawning multiple gossiping goroutines for the
// peer.
func (conR *Reactor) AddPeer(peer p2p.Peer) {
	if peer.ID() == conR.localPeerID {
		conR.sendNewRoundStepMessage(peer)
	}
}

// RemovePeer is a noop.
func (conR *Reactor) RemovePeer(p2p.Peer, interface{}) {
}

// Receive implements Reactor
// NOTE: We process these messages even when we're block_syncing.
// Messages affect either a peer state or the consensus state.
// Peer state updates can happen in parallel, but processing of
// proposals, block parts, and votes are ordered by the receiveRoutine
// NOTE: blocks on consensus state for proposals, block parts, and votes
func (conR *Reactor) Receive(e p2p.Envelope) {

}

// SetEventBus sets event bus.
func (conR *Reactor) SetEventBus(b *types.EventBus) {
}

// WaitSync returns whether the consensus reactor is waiting for state/block sync.
func (conR *Reactor) WaitSync() bool {
	return true
}

// func (conR *Reactor) broadcastNewRoundStepMessage(rs *cstypes.RoundState) {
// 	nrsMsg := makeRoundStepMessage()
// 	conR.Switch.Broadcast(p2p.Envelope{
// 		ChannelID: StateChannel,
// 		Message:   nrsMsg,
// 	})
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

// func (conR *Reactor) updateRoundStateRoutine() {
// 	t := time.NewTicker(100 * time.Microsecond)
// 	defer t.Stop()
// 	for range t.C {
// 		if !conR.IsRunning() {
// 			return
// 		}
// 		rs := conR.conS.GetRoundState()
// 		conR.mtx.Lock()
// 		conR.rs = rs
// 		conR.mtx.Unlock()
// 	}
// }

// func (conR *Reactor) getRoundState() *cstypes.RoundState {
// 	conR.mtx.RLock()
// 	defer conR.mtx.RUnlock()
// 	return conR.rs
// }

// String returns a string representation of the Reactor.
// NOTE: For now, it is just a hard-coded string to avoid accessing unprotected shared variables.
// TODO: improve!
func (conR *Reactor) String() string {
	// better not to access shared variables
	return "ConsensusReactor" // conR.StringIndented("")
}

//-----------------------------------------------------------------------------
// Messages

// Message is a message that can be sent and received on the Reactor
type Message interface {
	ValidateBasic() error
}

//func init() {
//	cmtjson.RegisterType(&NewRoundStepMessage{}, "tendermint/NewRoundStepMessage")
//}

//-------------------------------------

// NewRoundStepMessage is sent for every step taken in the ConsensusState.
// For every height/round/step transition
type NewRoundStepMessage struct {
	Height                int64
	Round                 int32
	Step                  cstypes.RoundStepType
	SecondsSinceStartTime int64
	LastCommitRound       int32
}

// ValidateBasic performs basic validation.
func (m *NewRoundStepMessage) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.Round < 0 {
		return errors.New("negative Round")
	}
	if !m.Step.IsValid() {
		return errors.New("invalid Step")
	}

	// NOTE: SecondsSinceStartTime may be negative

	// LastCommitRound will be -1 for the initial height, but we don't know what height this is
	// since it can be specified in genesis. The reactor will have to validate this via
	// ValidateHeight().
	if m.LastCommitRound < -1 {
		return errors.New("invalid LastCommitRound (cannot be < -1)")
	}

	return nil
}

// ValidateHeight validates the height given the chain's initial height.
func (m *NewRoundStepMessage) ValidateHeight(initialHeight int64) error {
	if m.Height < initialHeight {
		return fmt.Errorf("invalid Height %v (lower than initial height %v)",
			m.Height, initialHeight)
	}
	if m.Height == initialHeight && m.LastCommitRound != -1 {
		return fmt.Errorf("invalid LastCommitRound %v (must be -1 for initial height %v)",
			m.LastCommitRound, initialHeight)
	}
	if m.Height > initialHeight && m.LastCommitRound < 0 {
		return fmt.Errorf("LastCommitRound can only be negative for initial height %v",
			initialHeight)
	}
	return nil
}

// String returns a string representation.
func (m *NewRoundStepMessage) String() string {
	return fmt.Sprintf("[NewRoundStep H:%v R:%v S:%v LCR:%v]",
		m.Height, m.Round, m.Step, m.LastCommitRound)
}