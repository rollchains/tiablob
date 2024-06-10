package blockprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	protoblocktypes "github.com/cometbft/cometbft/proto/tendermint/types"

	//"github.com/rollchains/tiablob/celestia-node/blob"
	"github.com/rollchains/tiablob/celestia-node/share"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/relayer"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/celestia"
	"github.com/rollchains/tiablob/tiasync/blocksync/blockprovider/local"
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
	genTime time.Time

	store *store.BlockStore

	//blockCache map[int64]*protoblocktypes.Block

	mtx cmtsync.Mutex
}

func NewBlockProvider(store *store.BlockStore, celestiaHeight int64, celestiaCfg *relayer.CelestiaConfig, genTime time.Time) *BlockProvider {
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
		celestiaHeight: celestiaHeight,
		celestiaProvider: celestiaProvider,
		localProvider: localProvider,
		genTime: genTime,

		nodeRpcUrl:        celestiaCfg.NodeRpcURL,
		nodeAuthToken:     celestiaCfg.NodeAuthToken,
		celestiaNamespace: celestiaNamespace,
		celestiaChainID:   celestiaCfg.ChainID,

		store: store,
		
		//blockCache: make(map[int64]*protoblocktypes.Block),
	}
}

func (bp *BlockProvider) SetLogger(l log.Logger) {
	bp.logger = l
}

func (bp *BlockProvider) Start() {
	bp.logger.Debug("Block Provider Start()")
	ctx := context.Background()

	// first query for a celestia DA light client, use that height
	if bp.celestiaHeight == 0 {
		clientState, err := bp.localProvider.QueryCelestiaClientState(ctx)
		if err != nil {
			panic(err)
		} else {
			if clientState.LatestHeight.RevisionHeight > 0 {
				bp.celestiaHeight = int64(clientState.LatestHeight.RevisionHeight)
			}
		}
		bp.logger.Debug("Client state latest height: ", "height", clientState.LatestHeight.RevisionHeight)
	}

	// if still 0, get an estimated start height using genesis time
	if bp.celestiaHeight == 0 {
		celestiaHeight, err := bp.celestiaProvider.GetStartingCelestiaHeight(ctx, bp.genTime)
		if err != nil {
			panic(err)
		}
		bp.celestiaHeight = celestiaHeight
	}

	bp.logger.Debug("Starting to query celestia at height", "height", bp.celestiaHeight)

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			bp.queryCelestia(ctx)
			timer.Reset(10 * time.Second)
		}
	}

}

func (bp *BlockProvider) GetBlock(height int64) *protoblocktypes.Block {
	bp.logger.Debug("bp GetBlock()", "height", height)
	//return bp.blockCache[height]
	// TODO: we can just check the store's height, if less than requested height, it doesn't exist (seq vs non-seq)
	return bp.store.LoadBlock(height)
}

// func (bp *BlockProvider) GetHeight() int64 {
// 	bp.logger.Debug("bp GetHeight()", "height", len(bp.blockCache))
// 	return int64(len(bp.blockCache))
// }

func (bp *BlockProvider) queryCelestia(ctx context.Context) {
	bp.logger.Debug("bp queryCelestia()")
	celestiaNodeClient, err := cn.NewClient(ctx, bp.nodeRpcUrl, bp.nodeAuthToken)
	if err != nil {
		fmt.Println("creating celestia node client", "error", err)
		//r.logger.Error("creating celestia node client", "error", err)
		return
	}
	defer celestiaNodeClient.Close()

	celestiaLatestHeight, err := bp.celestiaProvider.QueryLatestHeight(ctx)
	if err != nil {
		fmt.Println("querying latest height from Celestia", "error", err)
		//r.logger.Error("querying latest height from Celestia", "error", err)
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
			fmt.Println("Celestia node blob getall", "height", queryHeight, "error", err)
			//r.logger.Error("Celestia node blob getall", "height", queryHeight, "error", err)
			return
		}

		for _, mBlob := range blobs {
			var blobBlockProto protoblocktypes.Block
			err := blobBlockProto.Unmarshal(mBlob.GetData())
			if err != nil {
				fmt.Println("blob unmarshal", "note", "may be a namespace collision", "height", queryHeight, "error", err)
				//r.logger.Info("blob unmarshal", "note", "may be a namespace collision", "height", queryHeight, "error", err)
			} else {
				rollchainBlockHeight := blobBlockProto.Header.Height
				bp.logger.Debug("bp adding block", "height", rollchainBlockHeight)
				bp.store.SaveBlock(celestiaLatestHeight, rollchainBlockHeight, mBlob.GetData())
				//bp.blockCache[rollchainBlockHeight] = &blobBlockProto
			}
		}
		
		bp.celestiaHeight = queryHeight
	}
}