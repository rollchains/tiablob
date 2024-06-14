package blockprovider

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
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

	"github.com/rollchains/tiablob/celestia-node/share"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/relayer"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/celestia"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/local"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/trustedrpc"
	"github.com/rollchains/tiablob/tiasync/store"
)

type BlockProvider struct {
	celestiaHeight int64
	logger log.Logger

	nodeRpcUrl string
	nodeAuthToken string
	celestiaChainID              string
	celestiaNamespace            appns.Namespace

	celestiaProvider *celestia.CosmosProvider
	localProvider    *local.CosmosProvider
	genDoc *cmttypes.GenesisDoc
	cmtConfig *cfg.Config

	genState sm.State
	clientCtx client.Context
	store *store.BlockStore
	
	valSet *cmttypes.ValidatorSet
	mtx cmtsync.Mutex
}

func NewBlockProvider(state sm.State, store *store.BlockStore, celestiaCfg *relayer.CelestiaConfig,
	genDoc *cmttypes.GenesisDoc, clientCtx client.Context, cmtConfig *cfg.Config) *BlockProvider {
	celestiaProvider, err := celestia.NewProvider(celestiaCfg.AppRpcURL, celestiaCfg.AppRpcTimeout)
	if err != nil {
		panic(err)
	}

	localProvider, err := local.NewProvider()
	if err != nil {
		panic(err)
	}

	//if cfg.OverrideNamespace != "" {
		celestiaNamespace := appns.MustNewV0([]byte(celestiaCfg.OverrideNamespace))
	//}

	return &BlockProvider{
		celestiaProvider: celestiaProvider,
		localProvider: localProvider,
		genDoc: genDoc,
		clientCtx: clientCtx,
		cmtConfig: cmtConfig,

		nodeRpcUrl:        celestiaCfg.NodeRpcURL,
		nodeAuthToken:     celestiaCfg.NodeAuthToken,
		celestiaNamespace: celestiaNamespace,
		celestiaChainID:   celestiaCfg.ChainID,

		genState: state,
		store: store,
	}
}

func (bp *BlockProvider) SetLogger(l log.Logger) {
	bp.logger = l
}

func (bp *BlockProvider) Start() {
	bp.logger.Info("Block Provider Start()", "celestia height", bp.celestiaHeight)
	ctx := context.Background()

	// Get the last height queried and if available, redo that query to ensure we got everything
	// Only perform this on Start() since we could have cleared it if the store was emptied
	bp.celestiaHeight = bp.store.LastCelestiaHeightQueried()-1

	// first query for a celestia DA light client, use that height (i.e. coming from state sync)
	if bp.celestiaHeight <= 0 {
		clientState, err := bp.localProvider.QueryCelestiaClientState(ctx)
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

	bp.queryLocalValSet(ctx)
	bp.queryCelestia(ctx)
	timer := time.NewTimer(5 * time.Second) // TODO: make configurable
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			bp.queryLocalValSet(ctx)
			bp.queryCelestia(ctx)
			timer.Reset(5 * time.Second)
		}
	}

}

func (bp *BlockProvider) GetVerifiedBlock(height int64) *protoblocktypes.Block {
	bp.logger.Info("bp GetBlock()", "height", height)
	if bp.store.Height()-1 < height {
		return nil
	}

	block1Meta := bp.store.LoadBlockMeta(height)
	block2Meta := bp.store.LoadBlockMeta(height+1)
	if block1Meta == nil || block2Meta == nil {
		return nil
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
		for block2Count := block2Meta.Count; block2Count > 0; block2Count--  {
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
				trustedServer := bp.cmtConfig.StateSync.RPCServers[0]
				if !strings.Contains(trustedServer, "://") {
					trustedServer = "http://" + trustedServer
				}
				trustedRpc, err := trustedrpc.NewProvider(trustedServer)
				if err != nil {
					bp.logger.Error("trusted rpc", "error", err)
					continue
				}
				// TODO: add a static context, don't create a new one each time
				valSet, err := trustedRpc.Validators(context.Background(), height)
				if err != nil {
					bp.logger.Error("trusted rpc validators", "error", err)
					continue
				}
				bp.valSet = &cmttypes.ValidatorSet{
					Validators: valSet.Validators,
				} 
				// Remove val set if we used a trusted node for the first two blocks after state sync
				defer func() {
					bp.valSet = nil
				}()
			}

			// otherwise, if valset is still nil, return although we shouldn't enter here
			if bp.valSet == nil {
				bp.logger.Info("Val set is nil")
				return nil
			}
			// TODO: pass in chain id
			// We are currently using the below method to verify 2/3 of the val set has signed the block.
			// Tiasync's val set is always at or near the current height, but we don't want to query it every block.
			// 2/3 is a little overkill for our needs which is to just to filter out blocks from namespace collisions
			// or malicious blocks. We may move towards using the light client verify with a trust level of 1/3.
			// If we use the light client verify method, we will also need to check that the txs, evidence, last commit 
			// hash correctly. The change may be warrented if the tiasync polling interval is high or 1/3 of the
			// validator set could frequently change. In either case, block sync will timeout more frequently and 
			// could impact catching up or staying at the expected height that Celestia has.
			err = bp.valSet.VerifyCommitLight(block1.Header.ChainID, block1ID, block1.Height, block2.LastCommit)
			if err == nil {
				bp.logger.Info("Block verified", "height", height)
				return block1Proto
			} else {
				bp.logger.Error("Block not verified", "height", height, "error", err)
			}
		}
	}

	return nil
}

// func (bp *BlockProvider) GetHeight() int64 {
// 	bp.logger.Debug("bp GetHeight()", "height", len(bp.blockCache))
// 	return int64(len(bp.blockCache))
// }

func (bp *BlockProvider) queryLocalValSet(ctx context.Context) {
	valSet, err := bp.clientCtx.Client.Validators(ctx, nil, nil, nil)
	if err == nil {
		bp.logger.Info("Updating val set")
		bp.mtx.Lock()
		bp.valSet = &cmttypes.ValidatorSet{
			Validators: valSet.Validators,
		}
		bp.mtx.Unlock()
	}
}

func (bp *BlockProvider) queryCelestia(ctx context.Context) {
	bp.logger.Info("bp queryCelestia()")
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

	bp.logger.Info("bp celestia latest height", "height", celestiaLatestHeight)
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

			// TODO: add some more checks like chain-id before saving
		
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
	for _, val := range bp.genState.Validators.Validators {
		valSet.Validators = append(valSet.Validators, val)
	}
	for _, genTx := range guGenesisState.GenTxs {
		tx, err := bp.clientCtx.TxConfig.TxJSONDecoder()(genTx)
		if err != nil {
			bp.logger.Error("Error decoding gentx", "tx", genTx)
			continue
		}

		msgs := tx.GetMsgs()
		for _, msg := range msgs {
			msgType := sdk.MsgTypeURL(msg)
			bp.logger.Info("Msg type", "type", msgType)
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
					Address: tmPubKey.Address(),
					PubKey: tmPubKey,
					VotingPower: createValMsg.Value.Amount.Int64(),
				})
			}
		}
	}

	sort.Sort(cmttypes.ValidatorsByVotingPower(valSet.Validators))

	return &valSet
}
