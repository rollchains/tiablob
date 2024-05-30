package tiasync

import (
	//"context"
	"errors"
	"fmt"
	//"net"

	cfg "github.com/cometbft/cometbft/config"
	sm "github.com/cometbft/cometbft/state"
	//"github.com/cometbft/cometbft/store"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/evidence"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"
	"github.com/cometbft/cometbft/version"
	bc "github.com/cometbft/cometbft/blocksync"
	//cs "github.com/cometbft/cometbft/consensus"

	//"github.com/rollchains/tiablob/tiasync/blocksync"
	"github.com/rollchains/tiablob/tiasync/store"
)

func createSwitch(config *cfg.Config,
	transport p2p.Transport,
	p2pMetrics *p2p.Metrics,
	peerFilters []p2p.PeerFilterFunc,
	//mempoolReactor p2p.Reactor,
	bcReactor p2p.Reactor,
	//stateSyncReactor *statesync.Reactor,
	//consensusReactor *cs.Reactor,
	//evidenceReactor *evidence.Reactor,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	p2pLogger log.Logger,
) *p2p.Switch {
	sw := p2p.NewSwitch(
		config.P2P,
		transport,
		p2p.WithMetrics(p2pMetrics),
		p2p.SwitchPeerFilters(peerFilters...),
	)
	sw.SetLogger(p2pLogger)
	//if config.Mempool.Type != cfg.MempoolTypeNop {
	//	sw.AddReactor("MEMPOOL", mempoolReactor)
	//}
	sw.AddReactor("BLOCKSYNC", bcReactor)
	//sw.AddReactor("CONSENSUS", consensusReactor)
	//sw.AddReactor("EVIDENCE", evidenceReactor)
	//sw.AddReactor("STATESYNC", stateSyncReactor)

	sw.SetNodeInfo(nodeInfo)
	sw.SetNodeKey(nodeKey)

	p2pLogger.Info("P2P Node ID", "ID", nodeKey.ID(), "file", config.NodeKeyFile())
	return sw
}

func createAddrBookAndSetOnSwitch(config *cfg.Config, sw *p2p.Switch,
	p2pLogger log.Logger, nodeKey *p2p.NodeKey,
) (pex.AddrBook, error) {
	addrBook := pex.NewAddrBook("ts"+config.P2P.AddrBookFile(), config.P2P.AddrBookStrict)
	addrBook.SetLogger(p2pLogger.With("tsbook", "ts"+config.P2P.AddrBookFile()))

	// Add ourselves to addrbook to prevent dialing ourselves
	if config.P2P.ExternalAddress != "" {
		addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeKey.ID(), config.P2P.ExternalAddress))
		if err != nil {
			return nil, fmt.Errorf("p2p.external_address is incorrect: %w", err)
		}
		addrBook.AddOurAddress(addr)
	}
	//if config.P2P.ListenAddress != "" {
		//addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeKey.ID(), config.P2P.ListenAddress))
		addr, err := p2p.NewNetAddressString(config.P2P.PersistentPeers)
		if err != nil {
			return nil, fmt.Errorf("p2p.laddr is incorrect: %w", err)
		}
		addrBook.AddOurAddress(addr)
	//}

	sw.SetAddrBook(addrBook)

	return addrBook, nil
}

func createTransport(
	config *cfg.Config,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	//proxyApp proxy.AppConns,
) (
	*p2p.MultiplexTransport,
	[]p2p.PeerFilterFunc,
) {
	var (
		mConnConfig = p2p.MConnConfig(config.P2P)
		transport   = p2p.NewMultiplexTransport(nodeInfo, *nodeKey, mConnConfig)
		connFilters = []p2p.ConnFilterFunc{}
		peerFilters = []p2p.PeerFilterFunc{}
	)

	if !config.P2P.AllowDuplicateIP {
		connFilters = append(connFilters, p2p.ConnDuplicateIPFilter())
	}

	// Filter peers by addr or pubkey with an ABCI query.
	// If the query return code is OK, add peer.
	/*if config.FilterPeers {
		connFilters = append(
			connFilters,
			// ABCI query for address filtering.
			func(_ p2p.ConnSet, c net.Conn, _ []net.IP) error {
				res, err := proxyApp.Query().Query(context.TODO(), &abci.RequestQuery{
					Path: fmt.Sprintf("/p2p/filter/addr/%s", c.RemoteAddr().String()),
				})
				if err != nil {
					return err
				}
				if res.IsErr() {
					return fmt.Errorf("error querying abci app: %v", res)
				}

				return nil
			},
		)

		peerFilters = append(
			peerFilters,
			// ABCI query for ID filtering.
			func(_ p2p.IPeerSet, p p2p.Peer) error {
				res, err := proxyApp.Query().Query(context.TODO(), &abci.RequestQuery{
					Path: fmt.Sprintf("/p2p/filter/id/%s", p.ID()),
				})
				if err != nil {
					return err
				}
				if res.IsErr() {
					return fmt.Errorf("error querying abci app: %v", res)
				}

				return nil
			},
		)
	}*/

	p2p.MultiplexTransportConnFilters(connFilters...)(transport)

	// Limit the number of incoming connections.
	//max := config.P2P.MaxNumInboundPeers + len(splitAndTrimEmpty(config.P2P.UnconditionalPeerIDs, ",", " "))
	max := 1
	p2p.MultiplexTransportMaxIncomingConnections(max)(transport)

	return transport, peerFilters
}

func makeNodeInfo(
	//config *cfg.Config,
	nodeKey *p2p.NodeKey,
	//txIndexer txindex.TxIndexer,
	genDoc *types.GenesisDoc,
	state sm.State,
) (p2p.DefaultNodeInfo, error) {
	//txIndexerStatus := "on"
	//if _, ok := txIndexer.(*null.TxIndex); ok {
	//	txIndexerStatus = "off"
	//}
	txIndexerStatus := "off"
	listenAddr := "tcp://0.0.0.0:26777"

	nodeInfo := p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.NewProtocolVersion(
			version.P2PProtocol, // global
			state.Version.Consensus.Block,
			state.Version.Consensus.App,
		),
		DefaultNodeID: nodeKey.ID(),
		Network:       genDoc.ChainID,
		Version:       version.TMCoreSemVer,
		Channels: []byte{
			bc.BlocksyncChannel,
			//cs.StateChannel, cs.DataChannel, cs.VoteChannel, cs.VoteSetBitsChannel,
			//mempl.MempoolChannel,
			//evidence.EvidenceChannel,
			//statesync.SnapshotChannel, statesync.ChunkChannel,
		},
		Moniker: "tiasync",
		//Moniker: config.Moniker,
		Other: p2p.DefaultNodeInfoOther{
			TxIndex:    txIndexerStatus,
			RPCAddress: listenAddr,
			//RPCAddress: config.RPC.ListenAddress,
		},
	}

	//if config.P2P.PexReactor {
	//	nodeInfo.Channels = append(nodeInfo.Channels, pex.PexChannel)
	//}

	//lAddr := config.P2P.ExternalAddress

	//if lAddr == "" {
	//	lAddr = config.P2P.ListenAddress
	//}

	nodeInfo.ListenAddr = listenAddr
	//nodeInfo.ListenAddr = lAddr

	err := nodeInfo.Validate()
	return nodeInfo, err
}


func createAndStartProxyAppConns(clientCreator proxy.ClientCreator, logger log.Logger, metrics *proxy.Metrics) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(clientCreator, metrics)
	proxyApp.SetLogger(logger.With("tsmodule", "tsproxy"))
	//if err := proxyApp.Start(); err != nil {
	//	return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	//}
	return proxyApp, nil
}

func createEvidenceReactor(config *cfg.Config, dbProvider cfg.DBProvider,
	stateStore sm.Store, blockStore *store.BlockStore, logger log.Logger,
) (*evidence.Reactor, *evidence.Pool, error) {
	evidenceDB, err := dbProvider(&cfg.DBContext{ID: "tsevidence", Config: config})
	if err != nil {
		return nil, nil, err
	}
	evidenceLogger := logger.With("tsmodule", "tsevidence")
	evidencePool, err := evidence.NewPool(evidenceDB, stateStore, blockStore)
	if err != nil {
		return nil, nil, err
	}
	evidenceReactor := evidence.NewReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)
	return evidenceReactor, evidencePool, nil
}

// MetricsProvider returns a consensus, p2p and mempool Metrics.
//type MetricsProvider func(chainID string) (*p2p.Metrics, *sm.Metrics, *proxy.Metrics, *blocksync.Metrics)

// DefaultMetricsProvider returns Metrics build using Prometheus client library
// if Prometheus is enabled. Otherwise, it returns no-op Metrics.
// func DefaultMetricsProvider(config *cfg.InstrumentationConfig) MetricsProvider {
// 	return func(chainID string) (*p2p.Metrics, *sm.Metrics, *proxy.Metrics, *blocksync.Metrics) {
// 		if config.Prometheus {
// 			return p2p.PrometheusMetrics(config.Namespace, "chain_id", chainID),
// 				sm.PrometheusMetrics(config.Namespace, "chain_id", chainID),
// 				proxy.PrometheusMetrics(config.Namespace, "chain_id", chainID),
// 				blocksync.PrometheusMetrics(config.Namespace, "chain_id", chainID)
// 		}
// 		return p2p.NopMetrics(), sm.NopMetrics(), proxy.NopMetrics(), blocksync.NopMetrics()
// 	}
// }

func initDBs(config *cfg.Config, dbProvider cfg.DBProvider) (blockStore *store.BlockStore, stateDB dbm.DB, err error) {
	var blockStoreDB dbm.DB
	blockStoreDB, err = dbProvider(&cfg.DBContext{ID: "tsblockstore", Config: config})
	if err != nil {
		return
	}
	blockStore = store.NewBlockStore(blockStoreDB)

	stateDB, err = dbProvider(&cfg.DBContext{ID: "tsstate", Config: config})
	if err != nil {
		return
	}

	return
}


// GenesisDocProvider returns a GenesisDoc.
// It allows the GenesisDoc to be pulled from sources other than the
// filesystem, for instance from a distributed key-value store cluster.
type GenesisDocProvider func() (*types.GenesisDoc, error)

var genesisDocKey = []byte("genesisDoc")

// LoadStateFromDBOrGenesisDocProvider attempts to load the state from the
// database, or creates one using the given genesisDocProvider. On success this also
// returns the genesis doc loaded through the given provider.
func LoadStateFromDBOrGenesisDocProvider(
	stateDB dbm.DB,
	genesisDocProvider GenesisDocProvider,
) (sm.State, *types.GenesisDoc, error) {
	// Get genesis doc
	genDoc, err := loadGenesisDoc(stateDB)
	if err != nil {
		genDoc, err = genesisDocProvider()
		if err != nil {
			return sm.State{}, nil, err
		}

		err = genDoc.ValidateAndComplete()
		if err != nil {
			return sm.State{}, nil, fmt.Errorf("error in genesis doc: %w", err)
		}
		// save genesis doc to prevent a certain class of user errors (e.g. when it
		// was changed, accidentally or not). Also good for audit trail.
		if err := saveGenesisDoc(stateDB, genDoc); err != nil {
			return sm.State{}, nil, err
		}
	}
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
	if err != nil {
		return sm.State{}, nil, err
	}
	return state, genDoc, nil
}

// panics if failed to unmarshal bytes
func loadGenesisDoc(db dbm.DB) (*types.GenesisDoc, error) {
	b, err := db.Get(genesisDocKey)
	if err != nil {
		panic(err)
	}
	if len(b) == 0 {
		return nil, errors.New("genesis doc not found")
	}
	var genDoc *types.GenesisDoc
	err = cmtjson.Unmarshal(b, &genDoc)
	if err != nil {
		panic(fmt.Sprintf("Failed to load genesis doc due to unmarshaling error: %v (bytes: %X)", err, b))
	}
	return genDoc, nil
}

// panics if failed to marshal the given genesis document
func saveGenesisDoc(db dbm.DB, genDoc *types.GenesisDoc) error {
	b, err := cmtjson.Marshal(genDoc)
	if err != nil {
		return fmt.Errorf("failed to save genesis doc due to marshaling error: %w", err)
	}
	return db.SetSync(genesisDocKey, b)
}