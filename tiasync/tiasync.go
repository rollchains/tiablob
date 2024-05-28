package tiasync

import (
	"fmt"
	"time"
	"os"
	"strings"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/proxy"
	sm "github.com/cometbft/cometbft/state"
	mempl "github.com/cometbft/cometbft/mempool"
	"github.com/cometbft/cometbft/types"

	"github.com/rollchains/tiablob/tiasync/blocksync"
)


type Tiasync struct {
	bcReactor         p2p.Reactor       // for block-syncing
}

func NewTiasync(
	config *cfg.Config,
	logger log.Logger,
) (*Tiasync, error) {
	dbProvider := cfg.DefaultDBProvider
	genesisDocProvider := func() (*types.GenesisDoc, error) {
		jsonBlob, err := os.ReadFile(config.GenesisFile())
		if err != nil {
			return nil, fmt.Errorf("couldn't read GenesisDoc file: %w", err)
		}
		jsonBlobStr := string(jsonBlob)
		jsonBlobStr = strings.ReplaceAll(jsonBlobStr, "\"initial_height\":1", "\"initial_height\":\"1\"")
		fmt.Println(jsonBlobStr)
		genDoc, err := types.GenesisDocFromJSON([]byte(jsonBlobStr))
		if err != nil {
			return nil, fmt.Errorf("error reading GenesisDoc at %s: %w", config.GenesisFile(), err)
		}
		return genDoc, nil
		//return types.GenesisDocFromFile(config.GenesisFile())
	}
	metricsProvider := func(chainID string) (*sm.Metrics, *proxy.Metrics, *blocksync.Metrics) {
		if config.Instrumentation.Prometheus {
			return sm.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				proxy.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				blocksync.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID)
		}
		return sm.NopMetrics(), proxy.NopMetrics(), blocksync.NopMetrics()
	}
	clientCreator := proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir())


/*	jsonBlob, err := os.ReadFile(config.GenesisFile())
	if err != nil {
		return nil, fmt.Errorf("couldn't read GenesisDoc file: %w", err)
	}*/



	blockStore, stateDB, err := initDBs(config, dbProvider)
	if err != nil {
		return nil, err
	}

	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})

	state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)
	if err != nil {
		return nil, err
	}

	smMetrics, abciMetrics, bsMetrics := metricsProvider(genDoc.ChainID)
	
	// Create the proxyApp and establish connections to the ABCI app (consensus, mempool, query).
	proxyApp, err := createAndStartProxyAppConns(clientCreator, logger, abciMetrics)
	if err != nil {
		return nil, err
	}
	mempool := &mempl.NopMempool{} //, _ := createMempoolAndMempoolReactor(config, proxyApp, state, memplMetrics, logger)
	
	_, evidencePool, err := createEvidenceReactor(config, dbProvider, stateStore, blockStore, logger)
	if err != nil {
		return nil, err
	}
	
	// make block executor for consensus and blocksync reactors to execute blocks
	blockExec := sm.NewBlockExecutor(
		stateStore,
		logger.With("tsmodule", "tsstate"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
		blockStore,
		sm.BlockExecutorWithMetrics(smMetrics),
	)
	
	offlineStateSyncHeight := int64(0)
	if blockStore.Height() == 0 {
		offlineStateSyncHeight, err = blockExec.Store().GetOfflineStateSyncHeight()
		if err != nil && err.Error() != "value empty" {
			panic(fmt.Sprintf("failed to retrieve statesynced height from store %s; expected state store height to be %v", err, state.LastBlockHeight))
		}
	}

	bcReactor := blocksync.NewReactor(state.Copy(), blockExec, blockStore, true, bsMetrics, offlineStateSyncHeight)
	bcReactor.SetLogger(logger.With("tsmodule", "tsblocksync"))

	tiasync := &Tiasync{
		bcReactor: bcReactor,
	}

	return tiasync, nil

}

func (t *Tiasync) Start() {
	timer := time.NewTimer(time.Second*5)
	defer timer.Stop()
	for {
		select {
		case <- timer.C:
			timer.Reset(time.Second*5)
		}
	}

}