package blocksync

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	bcproto "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider"
	"github.com/rollchains/tiablob/tiasync/store"
)

const (
	// BlocksyncChannel is a channel for blocks and status updates (`BlockStore` height)
	BlocksyncChannel = byte(0x40)
)

// Reactor handles long-term catchup syncing.
type Reactor struct {
	p2p.BaseReactor

	store *store.BlockStore

	localPeerInBlockSync bool // Local peer is in block sync mode
	clientCtx            client.Context

	localPeerID p2p.ID

	blockProvider        *blockprovider.BlockProvider
	celestiaPollInterval time.Duration
}

// NewReactor returns new reactor instance.
func NewReactor(state sm.State, store *store.BlockStore, localPeerID p2p.ID,
	celestiaCfg *relayer.CelestiaConfig, genDoc *types.GenesisDoc,
	clientCtx client.Context, cmtConfig *cfg.Config, celestiaPollInterval time.Duration,
	celestiaNamespace string, chainID string,
) *Reactor {

	bcR := &Reactor{
		localPeerID:          localPeerID,
		store:                store,
		localPeerInBlockSync: false,
		clientCtx:            clientCtx,
		blockProvider:        blockprovider.NewBlockProvider(state, store, celestiaCfg, genDoc, clientCtx, cmtConfig, celestiaNamespace, chainID),
		celestiaPollInterval: celestiaPollInterval,
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
				Base:   int64(0),
				Height: math.MaxInt64 - 1, // Send max to keep int block sync mode, peer won't request >20 blocks at the same time
			},
		})
	}
}

// RemovePeer implements Reactor by removing peer from the pool.
func (bcR *Reactor) RemovePeer(peer p2p.Peer, _ interface{}) {
	if peer.ID() == bcR.localPeerID {
		// this is a persistent peer, delay 100ms on removal due to reconnection speed
		// cometbft's block pool can still be cleaning up old peer
		bcR.Logger.Info("Removing local peer and delaying 100ms")
		time.Sleep(time.Millisecond * 100)
	}
}

// respondToPeer loads a block and sends it to the requesting peer,
// if we have it. Otherwise, we'll respond saying we don't have it.
func (bcR *Reactor) respondToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	block, commit := bcR.blockProvider.GetVerifiedBlock(msg.Height)
	if block == nil {
		bcR.Logger.Info("Peer asking for a block we don't have", "src", src, "height", msg.Height)
		return src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.NoBlockResponse{Height: msg.Height},
		})
	}

	// Ask local peer for their height for pruning (every 10 blocks sent)
	if msg.Height%10 == 0 {
		src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.StatusRequest{},
		})
	}

	var extCommit *types.ExtendedCommit
	voteExtensionEnableHeight := bcR.blockProvider.GetVoteExtensionsEnableHeight()
	if voteExtensionEnableHeight != 0 && voteExtensionEnableHeight <= msg.Height {
		bcR.Logger.Info("Vote extensions enabled, mocking extended commit")
		extCommit = &types.ExtendedCommit{
			Height: commit.Height,
			BlockID: commit.BlockID,
			Round: commit.Round,
		}
		for _, sig := range commit.Signatures {
			extCommitSig := types.ExtendedCommitSig{
				CommitSig: sig,
			}
			if sig.BlockIDFlag == types.BlockIDFlagCommit {
				extCommitSig.ExtensionSignature = sig.Signature
			}
			extCommit.ExtendedSignatures = append(extCommit.ExtendedSignatures, extCommitSig)
		}
	} else {
		bcR.Logger.Info("Vote extensions not enabled")
	}

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
			if !bcR.localPeerInBlockSync {
				// At this point, we start querying celestia. We don't start when the reactor is created because of state sync.
				// We know our local node has entered block sync,
				// and if we state sync'd, we will have our celestia da light client state with a latest height to start querying from.
				bcR.localPeerInBlockSync = true
				ctx := context.Background()
				res2, err := bcR.clientCtx.Client.ABCIInfo(ctx)
				if err != nil {
					bcR.localPeerInBlockSync = false
					bcR.Logger.Info("error get latest height2", "error", err)
					return
				} else {
					bcR.Logger.Info("queried abci info,  latest block height", "height", res2.Response.LastBlockHeight)
				}
				pruned, err := bcR.store.PruneBlocks(res2.Response.LastBlockHeight, true)
				if err != nil {
					bcR.Logger.Error("Error pruning blocks on startup", "error", err)
				}
				bcR.Logger.Info("blocks pruned on startup", "pruned", pruned)
				if bcR.store.IsEmpty() {
					bcR.store.SetInitialState(res2.Response.LastBlockHeight)
				}
				go bcR.blockProvider.Start(bcR.celestiaPollInterval)
			}
			bcR.respondToPeer(msg, e.Src)
		}
	case *bcproto.StatusRequest:
		// Send local peer our state.
		if e.Src.ID() == bcR.localPeerID {
			bcR.Logger.Debug("block sync Receive StatusRequest")
			e.Src.TrySend(p2p.Envelope{
				ChannelID: BlocksyncChannel,
				Message: &bcproto.StatusResponse{
					Height: math.MaxInt64 - 1, // Send max to keep int block sync mode, peer won't request >20 blocks at the same time
					Base:   int64(0),
				},
			})
		}
	case *bcproto.StatusResponse:
		if e.Src.ID() == bcR.localPeerID {
			bcR.Logger.Info("block sync Receive StatusResponse", "height", msg.Height, "base", msg.Base)
			pruned, err := bcR.store.PruneBlocks(msg.Height, false)
			if err != nil {
				bcR.Logger.Error("Error pruning blocks", "error", err)
			} else {
				bcR.Logger.Info("blocks pruned", "pruned", pruned)
			}
		}
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
