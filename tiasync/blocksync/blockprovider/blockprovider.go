package blockprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	cmtmath "github.com/cometbft/cometbft/libs/math"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	protoblocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/client"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gutypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/celestia-node/share"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	lc "github.com/rollchains/celestia-da-light-client"
	"github.com/rollchains/tiablob/relayer"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/celestia"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/trustedrpc"
	"github.com/rollchains/tiablob/tiasync/store"

	contypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
)

type BlockProvider struct {
	celestiaHeight int64
	logger         log.Logger

	nodeRpcUrl        string
	nodeAuthToken     string
	celestiaNamespace appns.Namespace
	chainID           string

	celestiaProvider *celestia.CosmosProvider
	genDoc           *cmttypes.GenesisDoc
	cmtConfig        *cfg.Config
	consensusQuerier contypes.QueryClient

	genState  sm.State
	clientCtx client.Context
	store     *store.BlockStore

	valSet         *cmttypes.ValidatorSet
	veEnableHeight int64
	mtx            cmtsync.Mutex
}

func NewBlockProvider(
	state sm.State,
	store *store.BlockStore,
	celestiaCfg *relayer.CelestiaConfig,
	genDoc *cmttypes.GenesisDoc,
	clientCtx client.Context,
	cmtConfig *cfg.Config,
	celestiaNamespaceStr string,
	chainID string,
) *BlockProvider {
	celestiaProvider, err := celestia.NewProvider(celestiaCfg.AppRpcURL, celestiaCfg.AppRpcTimeout)
	if err != nil {
		panic(err)
	}

	if celestiaCfg.OverrideNamespace != "" {
		celestiaNamespaceStr = celestiaCfg.OverrideNamespace
	}

	return &BlockProvider{
		celestiaProvider: celestiaProvider,
		genDoc:           genDoc,
		clientCtx:        clientCtx,
		cmtConfig:        cmtConfig,
		consensusQuerier: contypes.NewQueryClient(clientCtx),

		nodeRpcUrl:        celestiaCfg.NodeRpcURL,
		nodeAuthToken:     celestiaCfg.NodeAuthToken,
		celestiaNamespace: appns.MustNewV0([]byte(celestiaNamespaceStr)),
		chainID:           chainID,

		genState: state,
		store:    store,

		veEnableHeight: getInitialVoteExtensionEnableHeight(genDoc, cmtConfig, state),
	}
}

func (bp *BlockProvider) SetLogger(l log.Logger) {
	bp.logger = l
}

func (bp *BlockProvider) Start(celestiaPollInterval time.Duration) {
	bp.logger.Info("Block Provider Start()", "celestia height", bp.celestiaHeight)
	ctx := context.Background()

	// Get the last height queried and if available, redo that query to ensure we got everything
	// Only perform this on Start() since we could have cleared it if the store was emptied
	bp.celestiaHeight = bp.store.LastCelestiaHeightQueried() - 1

	// first query for a celestia DA light client, use that height (i.e. coming from state sync)
	if bp.celestiaHeight <= 0 {
		clientState, err := bp.QueryCelestiaClientState(ctx)
		if err != nil {
			panic(err)
		} else {
			if clientState.LatestHeight.RevisionHeight > 0 {
				bp.celestiaHeight = int64(clientState.LatestHeight.RevisionHeight)
			}
		}
		bp.logger.Info("Client state latest height: ", "height", clientState.LatestHeight.RevisionHeight)
	}

	// if still 0, get an estimated start height using genesis time
	if bp.celestiaHeight <= 0 {
		celestiaHeight, err := bp.celestiaProvider.GetStartingCelestiaHeight(ctx, bp.genDoc.GenesisTime)
		if err != nil {
			panic(err)
		}
		bp.celestiaHeight = celestiaHeight
	}

	bp.logger.Info("Starting to query celestia at height", "height", bp.celestiaHeight)

	bp.queryLocalNode(ctx)
	bp.queryCelestia(ctx)
	timer := time.NewTimer(celestiaPollInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			bp.queryLocalNode(ctx)
			bp.queryCelestia(ctx)
			timer.Reset(celestiaPollInterval)
		}
	}

}

func (bp *BlockProvider) GetVerifiedBlock(height int64) (*protoblocktypes.Block, *cmttypes.Commit) {
	bp.logger.Info("bp GetBlock()", "height", height)
	if bp.store.Height()-1 < height {
		return nil, nil
	}

	block1Meta := bp.store.LoadBlockMeta(height)
	block2Meta := bp.store.LoadBlockMeta(height + 1)
	if block1Meta == nil || block2Meta == nil {
		return nil, nil
	}

	for block1Count := block1Meta.Count; block1Count > 0; block1Count-- {
		block1Proto := bp.store.LoadBlock(height, block1Count-1)
		block1, err := cmttypes.BlockFromProto(block1Proto)
		if err != nil {
			continue
		}
		block1Parts, err := block1.MakePartSet(cmttypes.BlockPartSizeBytes)
		if err != nil {
			continue
		}
		block1PartSetHeader := block1Parts.Header()
		block1ID := cmttypes.BlockID{Hash: block1.Hash(), PartSetHeader: block1PartSetHeader}
		for block2Count := block2Meta.Count; block2Count > 0; block2Count-- {
			block2Proto := bp.store.LoadBlock(height+1, block2Count-1)
			block2, err := cmttypes.BlockFromProto(block2Proto)
			if err != nil {
				continue
			}

			bp.mtx.Lock()
			defer bp.mtx.Unlock()

			// if valSet is nil, we have blocks, and those blocks are w/in 20 blocks of genesis initial height, get genesis val set
			if bp.valSet == nil && height <= bp.genState.InitialHeight+20 {
				bp.valSet = bp.getGenesisValidatorSet()
			}

			// if valSet is nil, we have blocks, states sync is enabled, and we are w/in 1 block of the base, get val set from trusted rpc
			if bp.valSet == nil && bp.cmtConfig.StateSync.Enable && height <= bp.store.Base()+2 {
				bp.getValSetFromTrustedRpc(height)
				// Remove val set if we used a trusted node for the first two blocks after state sync
				defer func() {
					bp.valSet = nil
				}()
			}

			// otherwise, if valset is still nil, return although we shouldn't enter here
			if bp.valSet == nil {
				bp.logger.Info("Val set is nil")
				return nil, nil
			}

			// Verify that the block id of the block being verified matches that of the next block's last commit's block id
			// This verifies the txs, evidence, and last commit.
			if !block1ID.Equals(block2.LastCommit.BlockID) {
				bp.logger.Info("Block being verified does not match the next block's last commit block id, note: not an error", "height", block1.Height)
				continue
			}

			// Verify that 1/3 of our known val set has signed this block, this is to filter out invalid blocks due to namespace collisions or maliciousness.
			// After delivery via block sync, our local node will still verify 2/3 of the current val has signed it.
			// We use VerifyCommitLightTrusting instead of VerifyCommitLight in order to set the trusting level to 1/3, not require the val set size to
			// exactly match commit's signature size, and to look up signatures by address rather than index (val set doesn't necessarily correspond
			// with the val set that signed the block).
			// Most importantly, we do not need to have the latest val set. Since tiasync does not apply the blocks, we would need to query
			// the local node for the latest val set every block. This would result in block sync timing out every block (15 sec), slowing down syncing to
			// below the validator network's block production rate, and steadily getting further and further behind.
			err = bp.valSet.VerifyCommitLightTrusting(block1.Header.ChainID, block2.LastCommit, cmtmath.Fraction{Numerator: 1, Denominator: 3})
			if err == nil {
				bp.logger.Debug("Block verified w/ 1/3 of trusted val set", "height", height)
				return block1Proto, block2.LastCommit
			} else {
				bp.logger.Info("Block not verified, note: not an error", "height", height, "error", err)
			}
		}
	}

	return nil, nil
}

func (bp *BlockProvider) queryLocalNode(ctx context.Context) {
	valSet, err := bp.clientCtx.Client.Validators(ctx, nil, nil, nil)
	if err == nil {
		bp.logger.Debug("Updating val set")
		bp.mtx.Lock()
		bp.valSet = &cmttypes.ValidatorSet{
			Validators: valSet.Validators,
		}
		bp.mtx.Unlock()
	}

	paramsResp, err := bp.consensusQuerier.Params(ctx, &contypes.QueryParamsRequest{})
	if err == nil {
		bp.mtx.Lock()
		bp.logger.Info("vote extensions enable height", "height", paramsResp.Params.Abci.VoteExtensionsEnableHeight)
		bp.veEnableHeight = paramsResp.Params.Abci.VoteExtensionsEnableHeight
		bp.mtx.Unlock()
	}
}

func (bp *BlockProvider) queryCelestia(ctx context.Context) {
	bp.logger.Debug("bp queryCelestia()")
	celestiaNodeClient, err := cn.NewClient(ctx, bp.nodeRpcUrl, bp.nodeAuthToken)
	if err != nil {
		bp.logger.Error("creating celestia node client", "error", err)
		return
	}
	defer celestiaNodeClient.Close()

	celestiaLatestHeight, err := bp.celestiaProvider.QueryLatestHeight(ctx)
	if err != nil {
		bp.logger.Error("querying latest height from Celestia", "error", err)
		return
	}

	bp.logger.Debug("bp celestia latest height", "height", celestiaLatestHeight)
	for queryHeight := bp.celestiaHeight + 1; queryHeight < celestiaLatestHeight; queryHeight++ {
		// get the namespace blobs from that height
		blobs, err := celestiaNodeClient.Blob.GetAll(ctx, uint64(queryHeight), []share.Namespace{bp.celestiaNamespace.Bytes()})
		if err != nil {
			// this error just indicates we don't have a blob at this height
			if strings.Contains(err.Error(), "blob: not found") {
				bp.celestiaHeight = queryHeight
				continue
			}
			bp.logger.Error("Celestia node blob getall", "height", queryHeight, "error", err)
			return
		}

		for _, mBlob := range blobs {
			var blobBlockProto protoblocktypes.Block
			err := blobBlockProto.Unmarshal(mBlob.GetData())
			if err != nil {
				bp.logger.Info("blob unmarshal", "note", "may be a namespace collision", "height", queryHeight, "error", err)
				return
			}

			if bp.chainID != "" && blobBlockProto.Header.ChainID != bp.chainID {
				bp.logger.Info("blob's block does not have our chain id", "expected", bp.clientCtx.ChainID, "actual", blobBlockProto.Header.ChainID)
				return
			}

			rollchainBlockHeight := blobBlockProto.Header.Height
			bp.logger.Info("bp adding block", "height", rollchainBlockHeight)
			bp.store.SaveBlock(queryHeight, rollchainBlockHeight, mBlob.GetData())
		}

		bp.celestiaHeight = queryHeight
	}
}

func (bp *BlockProvider) getGenesisValidatorSet() *cmttypes.ValidatorSet {
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(bp.genDoc.AppState, &appState); err != nil {
		bp.logger.Error("unmarshal app state", "error", err)
		return nil
	}

	var guGenesisState gutypes.GenesisState
	bp.clientCtx.Codec.MustUnmarshalJSON(appState[gutypes.ModuleName], &guGenesisState)

	var valSet cmttypes.ValidatorSet
	valSet.Validators = append(valSet.Validators, bp.genState.Validators.Validators...)

	for _, genTx := range guGenesisState.GenTxs {
		tx, err := bp.clientCtx.TxConfig.TxJSONDecoder()(genTx)
		if err != nil {
			bp.logger.Error("Error decoding gentx", "tx", genTx)
			continue
		}

		msgs := tx.GetMsgs()
		for _, msg := range msgs {
			msgType := sdk.MsgTypeURL(msg)
			// the rest of this is pretty hacky...
			if msgType == "/cosmos.staking.v1beta1.MsgCreateValidator" {
				createValMsg, ok := msg.(*stakingtypes.MsgCreateValidator)
				if !ok {
					bp.logger.Error("casting msg to MsgCreateValidator not ok")
					continue
				}
				pubKey, ok := createValMsg.Pubkey.GetCachedValue().(cryptotypes.PubKey)
				if !ok {
					bp.logger.Error("casting pubkey not ok")
					continue
				}
				tmPubKey, err := cryptocodec.ToCmtPubKeyInterface(pubKey)
				if err != nil {
					bp.logger.Error("converting pubkey to tm pubkey", "error", err)
					continue
				}
				valSet.Validators = append(valSet.Validators, &cmttypes.Validator{
					Address:     tmPubKey.Address(),
					PubKey:      tmPubKey,
					VotingPower: createValMsg.Value.Amount.Int64(),
				})
			}
		}
	}

	sort.Sort(cmttypes.ValidatorsByVotingPower(valSet.Validators))

	return &valSet
}

func (bp *BlockProvider) QueryCelestiaClientState(ctx context.Context) (*lc.ClientState, error) {
	path := fmt.Sprintf("store/%s/key", tiablob.StoreKey)
	key := []byte(fmt.Sprintf("%s%s", tiablob.ClientStoreKey, lc.KeyClientState))

	res, err := bp.clientCtx.Client.ABCIQuery(ctx, path, key)
	if err != nil {
		return nil, err
	}

	var clientState lc.ClientState
	if err = bp.clientCtx.Codec.Unmarshal(res.Response.Value, &clientState); err != nil {
		return nil, err
	}

	return &clientState, nil
}

func (bp *BlockProvider) getValSetFromTrustedRpc(height int64) {
	ctx := context.Background()
	for _, trustedServer := range bp.cmtConfig.StateSync.RPCServers {
		if !strings.Contains(trustedServer, "://") {
			trustedServer = "http://" + trustedServer
		}
		trustedRpc, err := trustedrpc.NewProvider(trustedServer)
		if err != nil {
			bp.logger.Error("trusted rpc new provider", "error", err)
			continue
		}
		valSet, err := trustedRpc.Validators(ctx, height)
		if err != nil {
			bp.logger.Error("trusted rpc validators", "error", err)
			continue
		}
		bp.valSet = &cmttypes.ValidatorSet{
			Validators: valSet.Validators,
		}
		return
	}
}

func (bp *BlockProvider) GetVoteExtensionsEnableHeight() int64 {
	bp.mtx.Lock()
	defer bp.mtx.Unlock()
	return bp.veEnableHeight
}
