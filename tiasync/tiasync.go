package tiasync

import (
	"fmt"
	"net"
	"time"
	"os"
	"strings"
	"encoding/hex"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"
	"github.com/cometbft/cometbft/proxy"
	sm "github.com/cometbft/cometbft/state"
	mempl "github.com/cometbft/cometbft/mempool"
	"github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/store"
	"github.com/cometbft/cometbft/evidence"
	//rpccore "github.com/cometbft/cometbft/rpc/core"
	"github.com/cosmos/cosmos-sdk/server"
	cmtjson "github.com/cometbft/cometbft/libs/json"

	"github.com/rollchains/tiablob/tiasync/blocksync"
	"github.com/rollchains/tiablob/relayer"
)


type Tiasync struct {

	// config
	config        *cfg.Config
	genesisDoc    *types.GenesisDoc   // initial validator set
	privValidator types.PrivValidator // local node's validator key

	// network
	transport   *p2p.MultiplexTransport
	sw          *p2p.Switch  // p2p connections
	addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	// services
	//eventBus          *types.EventBus // pub/sub for services
	stateStore        sm.Store
	blockStore        *store.BlockStore // store the blockchain to disk
	bcReactor         p2p.Reactor       // for block-syncing
	//mempoolReactor    p2p.Reactor       // for gossipping transactions
	mempool           mempl.Mempool
	//stateSync         bool                    // whether the node should state sync on startup
	//stateSyncReactor  *statesync.Reactor      // for hosting and restoring state sync snapshots
	//stateSyncProvider statesync.StateProvider // provides state data for bootstrapping a node
	stateSyncGenesis  sm.State                // provides the genesis state for state sync
	//consensusState    *cs.State               // latest consensus state
	//consensusReactor  *cs.Reactor             // for participating in the consensus
	//pexReactor        *pex.Reactor            // for exchanging peer addresses
	evidencePool      *evidence.Pool          // tracking evidence
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

	tiasyncCfg := relayer.CelestiaConfigFromAppOpts(svrCtx.Viper)
	logger.Info("App opts: ", "Sync-from-celestia", tiasyncCfg.SyncFromCelestia, "chain id", tiasyncCfg.ChainID, "app rpc", tiasyncCfg.AppRpcURL)
	fmt.Println("App opts: ", "Sync-from-celestia", tiasyncCfg.SyncFromCelestia, "chain id", tiasyncCfg.ChainID, "app rpc", tiasyncCfg.AppRpcURL)

	if !tiasyncCfg.SyncFromCelestia {
		return
	}
	ts, err := NewTiasync(cometCfg, logger)
	if err != nil {
		panic(err)
	}

	ts.Start()
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
	metricsProvider := func(chainID string) (*p2p.Metrics, *sm.Metrics, *proxy.Metrics, *blocksync.Metrics) {
		if config.Instrumentation.Prometheus {
			return p2p.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				sm.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				proxy.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID),
				blocksync.PrometheusMetrics(config.Instrumentation.Namespace, "chain_id", chainID)
		}
		return p2p.NopMetrics(), sm.NopMetrics(), proxy.NopMetrics(), blocksync.NopMetrics()
	}
	clientCreator := proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir())
	//nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	//if err != nil {
	//	return nil, fmt.Errorf("failed to load or gen node key %s: %w", config.NodeKeyFile(), err)
	//}
	nodeKeyJson := "{\"priv_key\":{\"type\":\"tendermint/PrivKeyEd25519\",\"value\":\"O+iKUF7hOstRRkyKUUVsj28o96iIR2nD/xg8vG0KLJDVaQz24rVhdD+kWWUlm+Q/+1feJnR5mUIPcojpIyKLzg==\"}}"
	nodeKey := new(p2p.NodeKey)
	if err := cmtjson.Unmarshal([]byte(nodeKeyJson), nodeKey); err != nil {
		panic(err)
	}
	
	fmt.Println("NodeKey priv key:", hex.EncodeToString(nodeKey.PrivKey.Bytes()))
	fmt.Println("Nodekey pub key:", hex.EncodeToString(nodeKey.PubKey().Bytes()))
	fmt.Println("Nodekey id:", nodeKey.ID())
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

	p2pMetrics, smMetrics, abciMetrics, bsMetrics := metricsProvider(genDoc.ChainID)
	
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

	nodeInfo, err := makeNodeInfo(nodeKey, genDoc, state)
	if err != nil {
		return nil, err
	}

	transport, peerFilters := createTransport(config, nodeInfo, nodeKey, proxyApp)

	p2pLogger := logger.With("tsmodule", "tsp2p")
	//sw := createSwitch(
	//	config, transport, p2pMetrics, peerFilters, mempoolReactor, bcReactor,
	//	stateSyncReactor, consensusReactor, evidenceReactor, nodeInfo, nodeKey, p2pLogger,
	//)
	sw := createSwitch(
		config, transport, p2pMetrics, peerFilters, bcReactor, nodeInfo, nodeKey, p2pLogger,
	)

	//err = sw.AddPersistentPeers(splitAndTrimEmpty(config.P2P.PersistentPeers, ",", " "))
	//if err != nil {
	//	return nil, fmt.Errorf("could not add peers from persistent_peers field: %w", err)
	//}
	addrBook, err := createAddrBookAndSetOnSwitch(config, sw, p2pLogger, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("could not create addrbook: %w", err)
	}

	tiasync := &Tiasync{
		config:        config,
		genesisDoc:    genDoc,
		//privValidator: privValidator,

		transport: transport,
		sw:        sw,
		addrBook:  addrBook,
		nodeInfo:  nodeInfo,
		nodeKey:   nodeKey,

		stateStore:       stateStore,
		blockStore:       blockStore,
		bcReactor:        bcReactor,
		//mempoolReactor:   mempoolReactor,
		mempool:          mempool,
		//consensusState:   consensusState,
		//consensusReactor: consensusReactor,
		//stateSyncReactor: stateSyncReactor,
		//stateSync:        stateSync,
		stateSyncGenesis: state, // Shouldn't be necessary, but need a way to pass the genesis state
		//pexReactor:       pexReactor,
		evidencePool:     evidencePool,
		proxyApp:         proxyApp,
		//txIndexer:        txIndexer,
		//indexerService:   indexerService,
		//blockIndexer:     blockIndexer,
		//eventBus:         eventBus,
		Logger: logger,
	}

	return tiasync, nil

}

func (t *Tiasync) Start() {
	//now := cmttime.Now()
	//genTime := n.genesisDoc.GenesisTime
	//if genTime.After(now) {
	//	t.Logger.Info("Genesis time is in the future. Sleeping until then...", "genTime", genTime)
	//	time.Sleep(genTime.Sub(now))
	//}

	// run pprof server if it is enabled
	// if n.config.RPC.IsPprofEnabled() {
	// 	n.pprofSrv = n.startPprofServer()
	// }

	// begin prometheus metrics gathering if it is enabled
	// if n.config.Instrumentation.IsPrometheusEnabled() {
	// 	n.prometheusSrv = n.startPrometheusServer()
	// }

	// Start the RPC server before the P2P server
	// so we can eg. receive txs for the first block
	//if n.config.RPC.ListenAddress != "" {
	//	listeners, err := t.startRPC()
	//	if err != nil {
	//		return err
	//	}
	//	t.rpcListeners = listeners
	//}

	// Start the transport.
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(t.nodeKey.ID(), "tcp://0.0.0.0:26777"))
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
	//err = n.sw.DialPeersAsync(splitAndTrimEmpty(n.config.P2P.PersistentPeers, ",", " "))
	//if err != nil {
	//	return fmt.Errorf("could not dial peers from persistent_peers field: %w", err)
	//}

	timer := time.NewTimer(time.Second*5)
	defer timer.Stop()
	for {
		select {
		case <- timer.C:
			timer.Reset(time.Second*5)
		}
	}

}


/*// ConfigureRPC makes sure RPC has all the objects it needs to operate.
func (t *Tiasync) ConfigureRPC() (*rpccore.Environment, error) {
	pubKey, err := n.privValidator.GetPubKey()
	if pubKey == nil || err != nil {
		return nil, fmt.Errorf("can't get pubkey: %w", err)
	}
	rpcCoreEnv := rpccore.Environment{
		//ProxyAppQuery:   n.proxyApp.Query(),
		//ProxyAppMempool: n.proxyApp.Mempool(),

		StateStore:     t.stateStore,
		BlockStore:     t.blockStore,
		EvidencePool:   t.evidencePool,
		//ConsensusState: t.consensusState,
		P2PPeers:       t.sw,
		P2PTransport:   t,
		PubKey:         pubKey,

		GenDoc:           n.genesisDoc,
		TxIndexer:        n.txIndexer,
		BlockIndexer:     n.blockIndexer,
		ConsensusReactor: n.consensusReactor,
		EventBus:         n.eventBus,
		Mempool:          n.mempool,

		Logger: n.Logger.With("module", "rpc"),

		Config: *n.config.RPC,
	}
	if err := rpcCoreEnv.InitGenesisChunks(); err != nil {
		return nil, err
	}
	return &rpcCoreEnv, nil
}

func (t *Tiasync) startRPC() ([]net.Listener, error) {
	env, err := t.ConfigureRPC()
	if err != nil {
		return nil, err
	}

	listenAddrs := splitAndTrimEmpty(n.config.RPC.ListenAddress, ",", " ")
	routes := env.GetRoutes()

	if n.config.RPC.Unsafe {
		env.AddUnsafeRoutes(routes)
	}

	config := rpcserver.DefaultConfig()
	config.MaxBodyBytes = n.config.RPC.MaxBodyBytes
	config.MaxHeaderBytes = n.config.RPC.MaxHeaderBytes
	config.MaxOpenConnections = n.config.RPC.MaxOpenConnections
	// If necessary adjust global WriteTimeout to ensure it's greater than
	// TimeoutBroadcastTxCommit.
	// See https://github.com/tendermint/tendermint/issues/3435
	if config.WriteTimeout <= n.config.RPC.TimeoutBroadcastTxCommit {
		config.WriteTimeout = n.config.RPC.TimeoutBroadcastTxCommit + 1*time.Second
	}

	// we may expose the rpc over both a unix and tcp socket
	listeners := make([]net.Listener, len(listenAddrs))
	for i, listenAddr := range listenAddrs {
		mux := http.NewServeMux()
		rpcLogger := n.Logger.With("module", "rpc-server")
		wmLogger := rpcLogger.With("protocol", "websocket")
		wm := rpcserver.NewWebsocketManager(routes,
			rpcserver.OnDisconnect(func(remoteAddr string) {
				err := n.eventBus.UnsubscribeAll(context.Background(), remoteAddr)
				if err != nil && err != cmtpubsub.ErrSubscriptionNotFound {
					wmLogger.Error("Failed to unsubscribe addr from events", "addr", remoteAddr, "err", err)
				}
			}),
			rpcserver.ReadLimit(config.MaxBodyBytes),
			rpcserver.WriteChanCapacity(n.config.RPC.WebSocketWriteBufferSize),
		)
		wm.SetLogger(wmLogger)
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		rpcserver.RegisterRPCFuncs(mux, routes, rpcLogger)
		listener, err := rpcserver.Listen(
			listenAddr,
			config.MaxOpenConnections,
		)
		if err != nil {
			return nil, err
		}

		var rootHandler http.Handler = mux
		if n.config.RPC.IsCorsEnabled() {
			corsMiddleware := cors.New(cors.Options{
				AllowedOrigins: n.config.RPC.CORSAllowedOrigins,
				AllowedMethods: n.config.RPC.CORSAllowedMethods,
				AllowedHeaders: n.config.RPC.CORSAllowedHeaders,
			})
			rootHandler = corsMiddleware.Handler(mux)
		}
		if n.config.RPC.IsTLSEnabled() {
			go func() {
				if err := rpcserver.ServeTLS(
					listener,
					rootHandler,
					n.config.RPC.CertFile(),
					n.config.RPC.KeyFile(),
					rpcLogger,
					config,
				); err != nil {
					n.Logger.Error("Error serving server with TLS", "err", err)
				}
			}()
		} else {
			go func() {
				if err := rpcserver.Serve(
					listener,
					rootHandler,
					rpcLogger,
					config,
				); err != nil {
					n.Logger.Error("Error serving server", "err", err)
				}
			}()
		}

		listeners[i] = listener
	}

	return listeners, nil
}
*/