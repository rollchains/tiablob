package tiasync

import (
	"fmt"
	"net"
	"time"
	"os"
	"strings"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"
	"github.com/cometbft/cometbft/proxy"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/rollchains/tiablob/tiasync/blocksync"
	"github.com/rollchains/tiablob/tiasync/store"
	"github.com/rollchains/tiablob/tiasync/statesync"
	"github.com/rollchains/tiablob/tiasync/mempool"
	"github.com/rollchains/tiablob/tiasync/consensus"
	"github.com/rollchains/tiablob/relayer"
)


type Tiasync struct {

	// config
	config        *cfg.Config
	celestiaCfg   *relayer.CelestiaConfig
	tiasyncCfg    *TiasyncConfig
	genesisDoc    *types.GenesisDoc   // initial validator set
	privValidator types.PrivValidator // local node's validator key

	// network
	transport   *p2p.MultiplexTransport
	sw          *p2p.Switch  // p2p connections
	addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	cometNodeKey     *p2p.NodeKey // our node privkey

	// services
	//eventBus          *types.EventBus // pub/sub for services
	stateStore        sm.Store
	blockStore        *store.BlockStore // store the blockchain to disk
	bcReactor         p2p.Reactor       // for block-syncing
	mempoolReactor    p2p.Reactor       // for gossipping transactions
	//mempool           mempl.Mempool
	stateSync         bool                    // whether the node should state sync on startup
	stateSyncReactor  *statesync.Reactor      // for hosting and restoring state sync snapshots
	//stateSyncProvider statesync.StateProvider // provides state data for bootstrapping a node
	stateSyncGenesis  sm.State                // provides the genesis state for state sync
	//consensusState    *cs.State               // latest consensus state
	consensusReactor  *consensus.Reactor             // for participating in the consensus
	pexReactor        *pex.Reactor            // for exchanging peer addresses
	//evidencePool      *evidence.Pool          // tracking evidence
	proxyApp          proxy.AppConns          // connection to the application
	rpcListeners      []net.Listener          // rpc servers
	//txIndexer         txindex.TxIndexer
	//blockIndexer      indexer.BlockIndexer
	//indexerService    *txindex.IndexerService
	//prometheusSrv     *http.Server
	//pprofSrv          *http.Server

	Logger log.Logger
}

func TiasyncRoutine(svrCtx *server.Context) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	
	cometCfg := svrCtx.Config
	logger.Info("Comet config:", "Moniker", cometCfg.Moniker, "TimeoutCommit", cometCfg.Consensus.TimeoutCommit, "RootDir", cometCfg.RootDir)
	fmt.Println("Comet config:", "Moniker", cometCfg.Moniker, "TimeoutCommit", cometCfg.Consensus.TimeoutCommit, "RootDir", cometCfg.RootDir)
	logger.Info("Comet config:", "TimeoutPrecommit", cometCfg.Consensus.TimeoutPrecommit, "TimeoutPrevote", cometCfg.Consensus.TimeoutPrevote, "TimeoutPropose", cometCfg.Consensus.TimeoutPropose)
	fmt.Println("Comet config:", "TimeoutPrecommit", cometCfg.Consensus.TimeoutPrecommit, "TimeoutPrevote", cometCfg.Consensus.TimeoutPrevote, "TimeoutPropose", cometCfg.Consensus.TimeoutPropose)
	logger.Info("P2P:", "AddrBookStrict", cometCfg.P2P.AddrBookStrict, "ExternalAddress", cometCfg.P2P.ExternalAddress, "PersistentPeers", cometCfg.P2P.PersistentPeers, "ListenAddress", cometCfg.P2P.ListenAddress, "Seeds", cometCfg.P2P.Seeds)
	fmt.Println("P2P:", "AddrBookStrict", cometCfg.P2P.AddrBookStrict, "ExternalAddress", cometCfg.P2P.ExternalAddress, "PersistentPeers", cometCfg.P2P.PersistentPeers, "ListenAddress", cometCfg.P2P.ListenAddress, "Seeds", cometCfg.P2P.Seeds)

	tiasyncCfg := TiasyncConfigFromAppOpts(svrCtx.Viper)
	celestiaCfg := relayer.CelestiaConfigFromAppOpts(svrCtx.Viper)
	logger.Info("App opts: ", "Sync-from-celestia/enable", tiasyncCfg.Enable, "chain id", celestiaCfg.ChainID, "app rpc", celestiaCfg.AppRpcURL)
	fmt.Println("App opts: ", "Sync-from-celestia/enable", tiasyncCfg.Enable, "chain id", celestiaCfg.ChainID, "app rpc", celestiaCfg.AppRpcURL)

	if !tiasyncCfg.Enable {
		return
	}
	ts, err := NewTiasync(cometCfg, &tiasyncCfg, &celestiaCfg, logger)
	if err != nil {
		panic(err)
	}

	ts.Start()
}

func NewTiasync(
	config *cfg.Config,
	tiasyncCfg *TiasyncConfig,
	celestiaCfg *relayer.CelestiaConfig,
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
	}
	metricsProvider := func(chainID string) (*p2p.Metrics, *blocksync.Metrics, *statesync.Metrics) {
		if config.Instrumentation.Prometheus {
			return p2p.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				blocksync.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				statesync.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID)
		}
		return p2p.NopMetrics(), blocksync.NopMetrics(), statesync.NopMetrics()
	}
	
	nodeKey, err := p2p.LoadOrGenNodeKey(tiasyncCfg.NodeKeyFile(config.BaseConfig))
	if err != nil {
		panic(err)
	}
	
	cometNodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	blockStore, stateDB, err := initDBs(config, dbProvider)
	if err != nil {
		return nil, err
	}

	state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)
	if err != nil {
		return nil, err
	}
	p2pMetrics, bsMetrics, ssMetrics := metricsProvider(genDoc.ChainID)
	
	// _, evidencePool, err := createEvidenceReactor(config, dbProvider, stateStore, blockStore, logger)
	// if err != nil {
	// 	return nil, err
	// }

	bcReactor := blocksync.NewReactor(blockStore, cometNodeKey.ID(), bsMetrics, celestiaCfg, genDoc.GenesisTime)
	bcReactor.SetLogger(logger.With("tsmodule", "tsblocksync"))

	stateSyncReactor := statesync.NewReactor(
		*config.StateSync,
		ssMetrics,
	)
	stateSyncReactor.SetLogger(logger.With("tsmodule", "tsstatesync"))

	nodeInfo, err := makeNodeInfo(tiasyncCfg, nodeKey, genDoc, state)
	if err != nil {
		return nil, err
	}

	transport, peerFilters := createTransport(config, nodeInfo, nodeKey)

	p2pLogger := logger.With("tsmodule", "tsp2p")
	
	mempoolReactor := mempool.NewReactor(config.Mempool, cometNodeKey.ID())
	mempoolLogger := logger.With("tsmodule", "tsmempool")
	mempoolReactor.SetLogger(mempoolLogger)

	consensusReactor := consensus.NewReactor(cometNodeKey.ID())
	consensusLogger := logger.With("tsmodule", "tsconsensus")
	consensusReactor.SetLogger(consensusLogger)

	sw := createSwitch(
		config, transport, p2pMetrics, peerFilters, mempoolReactor, bcReactor,
		stateSyncReactor, consensusReactor, nodeInfo, nodeKey, p2pLogger,
	)

	err = sw.AddPersistentPeers(getPersistentPeers(tiasyncCfg.UpstreamPeers, cometNodeKey, config.P2P.ListenAddress))
	if err != nil {
		return nil, fmt.Errorf("could not add peers from persistent_peers field: %w", err)
	}
	addrBook, err := createAddrBookAndSetOnSwitch(config, tiasyncCfg, sw, p2pLogger, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("could not create addrbook: %w", err)
	}

	pexReactor := createPEXReactorAndAddToSwitch(addrBook, config, sw, logger)

	tiasync := &Tiasync{
		config:        config,
		tiasyncCfg:    tiasyncCfg,
		genesisDoc:    genDoc,
		//privValidator: privValidator,

		transport: transport,
		sw:        sw,
		addrBook:  addrBook,
		nodeInfo:  nodeInfo,
		nodeKey:   nodeKey,

		cometNodeKey: cometNodeKey,

		//stateStore:       stateStore,
		blockStore:       blockStore,
		bcReactor:        bcReactor,
		mempoolReactor:   mempoolReactor,
		//mempool:          mempool,
		//consensusState:   consensusState,
		consensusReactor: consensusReactor,
		stateSyncReactor: stateSyncReactor,
		//stateSync:        stateSync,
		//stateSyncGenesis: state, // Shouldn't be necessary, but need a way to pass the genesis state
		pexReactor:       pexReactor,
		//evidencePool:     evidencePool,
		//proxyApp:         proxyApp,
		//txIndexer:        txIndexer,
		//indexerService:   indexerService,
		//blockIndexer:     blockIndexer,
		//eventBus:         eventBus,
		Logger: logger,
	}

	return tiasync, nil

}

func (t *Tiasync) Start() {
	// Start the transport.
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(t.nodeKey.ID(), t.tiasyncCfg.ListenAddress))
	if err != nil {
		panic(err)
	}
	if err := t.transport.Listen(*addr); err != nil {
		panic(err)
	}

	t.isListening = true

	// Start the switch (the P2P server).
	err = t.sw.Start()
	if err != nil {
		panic(err)
	}

	// Always connect to persistent peers
	err = t.sw.DialPeersAsync(getPersistentPeers(t.tiasyncCfg.UpstreamPeers, t.cometNodeKey, t.config.P2P.ListenAddress))
	if err != nil {
		panic(fmt.Errorf("could not dial peers from persistent_peers field: %w", err))
	}

	timer := time.NewTimer(time.Second*5)
	defer timer.Stop()
	for {
		select {
		case <- timer.C:
			timer.Reset(time.Second*5)
		}
	}

}