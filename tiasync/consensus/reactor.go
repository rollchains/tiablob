package consensus

import (
	"math"

	"github.com/cometbft/cometbft/p2p"
	cmtcons "github.com/cometbft/cometbft/proto/tendermint/consensus"
	sm "github.com/cometbft/cometbft/state"
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
func (conR *Reactor) OnStop() {}

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

// AddPeer implements Reactor by spawning multiple gossiping goroutines for the
// peer.
func (conR *Reactor) AddPeer(peer p2p.Peer) {
	if peer.ID() == conR.localPeerID {
		nrsMsg := defaultRoundStepMessage()
		peer.Send(p2p.Envelope{
			ChannelID: StateChannel,
			Message:   nrsMsg,
		})
	}
}

func defaultRoundStepMessage() (nrsMsg *cmtcons.NewRoundStep) {
	nrsMsg = &cmtcons.NewRoundStep{
		Height:                math.MaxInt64-1, // Local node should always be less than our height for tx propagation
		Round:                 0,
		Step:                  1,
		SecondsSinceStartTime: 1,
		LastCommitRound:       0,
	}
	return
}

// String returns a string representation of the Reactor.
// NOTE: For now, it is just a hard-coded string to avoid accessing unprotected shared variables.
// TODO: improve!
func (conR *Reactor) String() string {
	// better not to access shared variables
	return "ConsensusReactor" // conR.StringIndented("")
}