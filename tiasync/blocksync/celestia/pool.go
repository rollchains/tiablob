package celestia

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtsync "github.com/cometbft/cometbft/libs/sync"
	protoblocktypes "github.com/cometbft/cometbft/proto/tendermint/types"
	
	//"github.com/rollchains/tiablob/celestia-node/blob"
	appns "github.com/rollchains/tiablob/celestia/namespace"
	"github.com/rollchains/tiablob/celestia-node/share"
	cn "github.com/rollchains/tiablob/relayer/celestia-node"
	"github.com/rollchains/tiablob/relayer"
)

type BlockPool struct {
	celestiaHeight int64
	logger log.Logger

	//rollchainHeight int64

	nodeRpcUrl string
	nodeAuthToken string
	celestiaChainID              string
	celestiaNamespace            appns.Namespace

	celestiaProvider *CosmosProvider

	blockCache map[int64]*protoblocktypes.Block

	mtx cmtsync.Mutex
}

func NewBlockPool(celestiaHeight int64, celestiaCfg *relayer.CelestiaConfig, logger log.Logger) *BlockPool {
	celestiaProvider, err := NewProvider(celestiaCfg.AppRpcURL, celestiaCfg.AppRpcTimeout)
	if err != nil {
		panic(err)
	}

	//if cfg.OverrideNamespace != "" {
		celestiaNamespace := appns.MustNewV0([]byte(celestiaCfg.OverrideNamespace))
	//}

	return &BlockPool{
		celestiaHeight: celestiaHeight,
		logger: logger,
		//rollchainHeight: rollchainHeight,
		celestiaProvider: celestiaProvider,

		nodeRpcUrl:        celestiaCfg.NodeRpcURL,
		nodeAuthToken:     celestiaCfg.NodeAuthToken,
		celestiaNamespace: celestiaNamespace,
		celestiaChainID:   celestiaCfg.ChainID,
		
		blockCache: make(map[int64]*protoblocktypes.Block),
	}
}

func (bp *BlockPool) Start() {
	bp.logger.Debug("Block Pool Start()")
	ctx := context.Background()

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

func (bp *BlockPool) GetBlock(height int64) *protoblocktypes.Block {
	bp.logger.Debug("bp GetBlock()", "height", height)
	return bp.blockCache[height]
}

func (bp *BlockPool) GetHeight() int64 {
	bp.logger.Debug("bp GetHeight()", "height", len(bp.blockCache))
	return int64(len(bp.blockCache))
}

func (bp *BlockPool) queryCelestia(ctx context.Context) {
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
				bp.blockCache[rollchainBlockHeight] = &blobBlockProto
			}
		}
		
		bp.celestiaHeight = queryHeight
	}
}