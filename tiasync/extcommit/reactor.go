package extcommit

import (
	"fmt"
	"reflect"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtmath "github.com/cometbft/cometbft/libs/math"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/p2p"
	bcproto "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"

	"github.com/cosmos/gogoproto/proto"
)

const (
	// BlocksyncChannel is a channel for blocks and status updates (`BlockStore` height)
	BlocksyncChannel = byte(0x40)
)

// Reactor handles long-term catchup syncing.
type Reactor struct {
	p2p.BaseReactor

	// immutable
	state sm.State

	mtx            cmtsync.Mutex
	extCommitFound bool
	height         int64

	blockStoreDB dbm.DB
}

// NewReactor returns new reactor instance.
func NewReactor(state sm.State, logger log.Logger, blockStoreDB dbm.DB, height int64) *Reactor {

	ecR := &Reactor{
		state:        state,
		blockStoreDB: blockStoreDB,
		height:       height,
	}
	ecR.BaseReactor = *p2p.NewBaseReactor("Reactor", ecR)
	ecR.SetLogger(logger)

	// at this point
	// if extCommitFound == true, we already have the necessary extended commit to start up cometbft
	// if extCommitFound == false, we need to get it from a peer

	go ecR.requestRoutine()

	return ecR
}

func (ecR *Reactor) FoundStatus() bool {
	ecR.mtx.Lock()
	defer ecR.mtx.Unlock()
	return ecR.extCommitFound
}

// GetChannels implements Reactor
func (ecR *Reactor) GetChannels() []*p2p.ChannelDescriptor {
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

// AddPeer implements Reactor by sending our request to peer.
func (ecR *Reactor) AddPeer(peer p2p.Peer) {
	//	ecR.mtx.Lock()
	//	ecR.peers[peer.ID()] = peer
	//	defer ecR.mtx.Unlock()
	if !ecR.FoundStatus() {
		peer.Send(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message: &bcproto.BlockRequest{
				Height: ecR.height,
			},
		})
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (ecR *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {
	// ecR.mtx.Lock()
	// defer ecR.mtx.Unlock()
	// delete(ecR.peers, peer.ID())
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (ecR *Reactor) Receive(e p2p.Envelope) {
	if err := ValidateMsg(e.Message); err != nil {
		ecR.Logger.Error("Peer sent us invalid msg", "peer", e.Src, "msg", e.Message, "err", err)
		ecR.Switch.StopPeerForError(e.Src, err)
		return
	}

	ecR.Logger.Debug("Receive", "e.Src", e.Src, "chID", e.ChannelID, "msg", e.Message)

	switch msg := e.Message.(type) {
	case *bcproto.BlockRequest:
		ecR.Logger.Info("Peer asking for a block we don't have", "src", e.Src, "height", msg.Height)
		e.Src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.NoBlockResponse{Height: msg.Height},
		})
	case *bcproto.BlockResponse:
		ecR.mtx.Lock()
		defer ecR.mtx.Unlock()
		if ecR.extCommitFound {
			ecR.Logger.Info("Extended commit already found")
			return
		}
		if msg.Block.Header.Height != ecR.height {
			ecR.Logger.Error("Peer sent us wrong block", "peer", e.Src, "expected height", ecR.height, "actual height", msg.Block.Header.Height)
			return
		}
		bi, err := types.BlockFromProto(msg.Block)
		if err != nil {
			ecR.Logger.Error("Peer sent us invalid block", "peer", e.Src, "msg", e.Message, "err", err)
			ecR.Switch.StopPeerForError(e.Src, err)
			return
		}
		if msg.ExtCommit == nil {
			// extended commit must be included
			ecR.Logger.Error("Peer sent us nil extended commit", "peer", e.Src, "msg", e.Message, "err", err)
			ecR.Switch.StopPeerForError(e.Src, err)
			return
		}
		extCommit, err := types.ExtendedCommitFromProto(msg.ExtCommit)
		if err != nil {
			ecR.Logger.Error("failed to convert extended commit from proto", "peer", e.Src, "err", err)
			ecR.Switch.StopPeerForError(e.Src, err)
			return
		}
		for _, sig := range extCommit.ExtendedSignatures {
			// Check to ensure the extended commit has unique signatures for block and extensions
			if sig.BlockIDFlag == types.BlockIDFlagCommit && reflect.DeepEqual(sig.Signature, sig.ExtensionSignature) {
				ecR.Logger.Error("Peer sent us invalid extended commit", "peer", e.Src, "msg", e.Message, "err", err)
				ecR.Switch.StopPeerForError(e.Src, err)
				return
			}
		}

		commit := extCommit.ToCommit()
		err = ecR.state.Validators.VerifyCommitLightTrusting(bi.ChainID, commit, cmtmath.Fraction{Numerator: 1, Denominator: 3})
		if err != nil {
			ecR.Logger.Error("failed to verify block that came with extended commit", "peer", e.Src, "err", err)
			ecR.Switch.StopPeerForError(e.Src, err)
			return
		}

		extCommitBytes := mustEncode(msg.ExtCommit)
		if err := ecR.blockStoreDB.Set(calcExtCommitKey(ecR.height), extCommitBytes); err != nil {
			panic(err)
		}
		ecR.Logger.Info("Extended commit found", "height", ecR.height, "peer", e.Src)

		ecR.extCommitFound = true
		_ = ecR.Stop()
	case *bcproto.StatusRequest:
		e.Src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message: &bcproto.StatusResponse{
				Height: 0,
				Base:   0,
			},
		})
	case *bcproto.StatusResponse:
	case *bcproto.NoBlockResponse:
		ecR.Logger.Debug("Peer does not have requested block", "peer", e.Src, "height", msg.Height)
	default:
		ecR.Logger.Info(fmt.Sprintf("Unknown message type %v, not an error", reflect.TypeOf(msg)))
	}
}

func (ecR *Reactor) requestRoutine() {
	requestTicker := time.NewTicker(time.Second * 15)
	defer requestTicker.Stop()

	for {
		select {
		case <-ecR.Quit():
			return
		case <-requestTicker.C:
			if ecR.FoundStatus() {
				return
			}
			go ecR.broadcastBlockRequest(ecR.height)
		}
	}
}

// broadcastBlockRequest broadcasts a request for a block height
func (ecR *Reactor) broadcastBlockRequest(height int64) {
	ecR.Logger.Info("Extended commit not found yet, re-broadcasting block request. Consider adding more peers.", "height", ecR.height)
	ecR.Switch.Broadcast(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message:   &bcproto.BlockRequest{Height: height},
	})
}

// mustEncode proto encodes a proto.message and panics if fails
func mustEncode(pb proto.Message) []byte {
	bz, err := proto.Marshal(pb)
	if err != nil {
		panic(fmt.Errorf("unable to marshal: %w", err))
	}
	return bz
}

func calcExtCommitKey(height int64) []byte {
	return []byte(fmt.Sprintf("EC:%v", height))
}
