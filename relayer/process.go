package relayer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/rollchains/tiablob/celestia/appconsts"
	blobtypes "github.com/rollchains/tiablob/celestia/blob/types"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/relayer/cosmos"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	celestiaKeyRingName  = "blob"
	celestiaBech32Prefix = "celestia"
	celestiaBlobPostMemo = "Posted by tiablob https://rollchains.com"
)

// Relayer is responsible for posting new blocks to Celestia and relaying block proofs from Celestia via the current proposer
type Relayer struct {
	logger log.Logger

	provenHeights      chan uint64
	latestProvenHeight uint64

	commitHeights      chan uint64
	latestCommitHeight uint64

	pollInterval time.Duration

	blockProofCache      map[uint64][]byte
	blockProofCacheLimit int

	provider  *cosmos.CosmosProvider
	clientCtx client.Context

	celestiaChainID       string
	celestiaNamespace     appns.Namespace
	celestiaGasPrice      string
	celestiaGasAdjustment float64
}

// NewRelayer creates a new Relayer instance
func NewRelayer(
	logger log.Logger,
	celestiaRpcUrl string,
	celestiaNamespace appns.Namespace,
	celestiaGasPrice string,
	celestiaGasAdjustment float64,
	keyDir string,
	pollInterval time.Duration,
	blockProofCacheLimit int,
) (*Relayer, error) {
	provider, err := cosmos.NewProvider(celestiaRpcUrl, keyDir)
	if err != nil {
		return nil, err
	}

	return &Relayer{
		logger: logger,

		pollInterval: pollInterval,

		provenHeights: make(chan uint64, 10000),
		commitHeights: make(chan uint64, 10000),

		provider:              provider,
		celestiaNamespace:     celestiaNamespace,
		celestiaGasPrice:      celestiaGasPrice,
		celestiaGasAdjustment: celestiaGasAdjustment,

		blockProofCache:      make(map[uint64][]byte),
		blockProofCacheLimit: blockProofCacheLimit,
	}, nil
}

func (r *Relayer) SetClientContext(clientCtx client.Context) {
	r.clientCtx = clientCtx
}

// Start begins the relayer process
func (r *Relayer) Start(
	ctx context.Context,
	provenHeight uint64,
	commitHeight uint64,
) error {
	r.latestProvenHeight = provenHeight
	r.latestCommitHeight = commitHeight

	if err := r.provider.CreateKeystore(); err != nil {
		return err
	}

	chainID, err := r.provider.QueryChainID(ctx)
	if err != nil {
		return fmt.Errorf("error querying celestia chain ID: %w", err)
	}

	r.celestiaChainID = chainID

	timer := time.NewTimer(r.pollInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case height := <-r.provenHeights:
			r.updateHeight(height)
		case <-timer.C:
			r.checkForNewBlockProofs(ctx)
			timer.Reset(r.pollInterval)
		}
	}
}

// NotifyCommitHeight is called by the app to notify the relayer of the latest commit height
func (r *Relayer) NotifyCommitHeight(height uint64) {
	r.provenHeights <- height
}

// NotifyProvenHeight is called by the app to notify the relayer of the latest proven height
// i.e. the height of the highest incremental block that was proven to be posted to Celestia.
func (r *Relayer) NotifyProvenHeight(height uint64) {
	r.provenHeights <- height
}

func (r *Relayer) updateHeight(height uint64) {
	if height > r.latestProvenHeight {
		r.latestProvenHeight = height

		for h := range r.blockProofCache {
			if h <= height {
				// this height has been proven (either by us or another proposer), we can delete the cached proof
				delete(r.blockProofCache, h)
			}
		}
	}
}

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context) {
	// celestiaHeight, err := r.provider.QueryLatestHeight(ctx)
	// if err != nil {
	// 	r.logger.Error("Error querying latest height from Celestia", "error", err)
	// 	return
	// }

	//for i := r.latestProvenHeight + 1; i <= r.latestCommitHeight; i++ {

	// blob := blob.New(namespace.PayForBlobNamespace, []byte(fmt.Sprintf("TODO populate with block data for height i: %d", i)), 0)

	// // generate share commitment for height i
	// commitment, err := inclusion.CreateCommitment(blob, merkle.HashFromByteSlices, 64)
	// if err != nil {
	// 	r.logger.Error("error creating commitment for height", "height", i, "error", err)
	// 	break
	// }

	// // TODO confirm all of these details. Looks like the share commitment may not be able to be queried in the typical way
	// res, err := r.provider.QueryABCI(ctx, abci.RequestQuery{
	// 	Path:   "store/blob/key",
	// 	Data:   commitment,
	// 	Height: celestiaHeight,
	// 	Prove:  true,
	// })
	// if err != nil {
	// 	r.logger.Error("error querying share commitment proof for height", "height", i, "error", err)
	// 	break
	// }

	// if res.Code != 0 {
	// 	r.logger.Error("error querying share commitment proof for height", "height", i, "code", res.Code, "log", res.Log)
	// 	break
	// }

	// cache the proof
	//r.blockProofCache[i] = res.ProofOps.Ops
	//}
}

// Reconcile is intended to be called by the current proposing validator during PrepareProposal and will:
// - check the cache (no fetches here to minimize duration) if there are any new block proofs to be relayed from Celestia
// - if there are any block proofs to relay, generate a MsgUpdateClient along with the MsgProveBlock message(s) and return them in a tx for injection into the proposal.
// - if the Celestia light client is within 1/3 of the trusting period and there are no block proofs to relay, generate a MsgUpdateClient to update the light client and return it in a tx.
// - in a non-blocking goroutine, post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
func (r *Relayer) Reconcile(ctx sdk.Context) [][]byte {
	go r.postNextBlocks(ctx)

	// TODO check cache for new block proofs to relay
	return nil
}

// postNextBlocks will post the next block (or chunk of blocks) above the last proven height to Celestia (does not include the block being proposed).
// Skip the last n blocks to give time for the previous proposer's transaction to succeed.
func (r *Relayer) postNextBlocks(ctx sdk.Context) {
	// TODO post blocks since last proven height
	height := ctx.BlockHeight()

	if height <= 1 {
		return
	}

	if !r.provider.KeyExists(celestiaKeyRingName) {
		r.logger.Error("No Celestia key found, please add with `tiablob keys add` command")
		return
	}

	res, err := txtypes.NewServiceClient(r.clientCtx).GetBlockWithTxs(ctx, &txtypes.GetBlockWithTxsRequest{
		Height: height - 1,
	})
	if err != nil {
		r.logger.Error("Error getting block", "error", err)
		return
	}

	blockBz, err := res.Block.Marshal()
	if err != nil {
		r.logger.Error("Error marshaling block", "error", err)
		return
	}

	blob, err := blobtypes.NewBlob(r.celestiaNamespace, blockBz, appconsts.ShareVersionZero)
	if err != nil {
		r.logger.Error("Error building blob", "error", err)
		return
	}

	// 12000 required for feegrant
	gasLimit := blobtypes.DefaultEstimateGas([]uint32{uint32(len(blob.Data))}) + 12000

	signer, err := r.provider.ShowAddress(celestiaKeyRingName, "celestia")
	if err != nil {
		r.logger.Error("Error getting signer address", "error", err)
		return
	}

	msg, err := blobtypes.NewMsgPayForBlobs(signer, blob)
	if err != nil {
		r.logger.Error("Error building MsgPayForBlobs", "error", err)
		return
	}

	ws := r.provider.EnsureWalletState(celestiaKeyRingName)
	ws.Mu.Lock()
	defer ws.Mu.Unlock()

	seq, txBz, err := r.provider.Sign(
		ctx,
		ws,
		r.celestiaChainID,
		r.celestiaGasPrice,
		r.celestiaGasAdjustment,
		gasLimit,
		celestiaBech32Prefix,
		celestiaKeyRingName,
		[]sdk.Msg{msg},
		celestiaBlobPostMemo,
	)
	if err != nil {
		// Account sequence mismatch errors can happen on the simulated transaction also.
		if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
			ws.HandleAccountSequenceMismatchError(err)
		}

		r.logger.Error("Error signing blob tx", "error", err)
		return
	}

	blobTx, err := tmtypes.MarshalBlobTx(txBz, blob)
	if err != nil {
		r.logger.Error("Error marshaling blob tx", "error", err)
		return
	}

	if err := r.provider.Broadcast(ctx, seq, ws, blobTx); err != nil {
		if strings.Contains(err.Error(), legacyerrors.ErrWrongSequence.Error()) {
			ws.HandleAccountSequenceMismatchError(err)
		}

		r.logger.Error("Error broadcasting blob tx", "error", err)

		return
	}

	r.logger.Info("Posted block to Celestia", "height", height)
}
