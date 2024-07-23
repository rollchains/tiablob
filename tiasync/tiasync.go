package tiasync

import (
	"fmt"
	"os"
	"strings"

	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/tiasync/blocksync"
	"github.com/rollchains/tiablob/tiasync/consensus"
	"github.com/rollchains/tiablob/tiasync/mempool"
	"github.com/rollchains/tiablob/tiasync/statesync"
	"github.com/rollchains/tiablob/tiasync/store"
)

type Tiasync struct {

	// config
	cmtConfig   *cfg.Config
	celestiaCfg *relayer.CelestiaConfig
	tiasyncCfg  *TiasyncConfig
	genesisDoc  *types.GenesisDoc // initial validator set

	// network
	transport   *p2p.MultiplexTransport
	sw          *p2p.Switch  // p2p connections
	addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	cometNodeKey *p2p.NodeKey // our node privkey

	// services
	blockStore       *store.BlockStore  // store the blockchain to disk
	bcReactor        p2p.Reactor        // for block-syncing
	mempoolReactor   p2p.Reactor        // for gossipping transactions
	stateSyncReactor *statesync.Reactor // for hosting and restoring state sync snapshots
	consensusReactor *consensus.Reactor // for participating in the consensus
	pexReactor       *pex.Reactor       // for exchanging peer addresses

	Logger log.Logger
}

func TiasyncRoutine(svrCtx *server.Context, clientCtx client.Context, celestiaNamespace string) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	cometCfg := svrCtx.Config
	tiasyncCfg := TiasyncConfigFromAppOpts(svrCtx.Viper)
	celestiaCfg := relayer.CelestiaConfigFromAppOpts(svrCtx.Viper)

	if !tiasyncCfg.Enable {
		return
	}

	logger, err := cmtflags.ParseLogLevel(cometCfg.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		panic(err)
	}
	ts, err := NewTiasync(cometCfg, &tiasyncCfg, &celestiaCfg, logger, clientCtx, celestiaNamespace)
	if err != nil {
		panic(err)
	}

	ts.Start()

	// Run forever
	select {}
}

func NewTiasync(
	cmtConfig *cfg.Config,
	tiasyncCfg *TiasyncConfig,
	celestiaCfg *relayer.CelestiaConfig,
	logger log.Logger,
	clientCtx client.Context,
	celestiaNamespace string,
) (*Tiasync, error) {
	dbProvider := cfg.DefaultDBProvider
	genesisDocProvider := func() (*types.GenesisDoc, error) {
		jsonBlob, err := os.ReadFile(cmtConfig.GenesisFile())
		if err != nil {
			return nil, fmt.Errorf("couldn't read GenesisDoc file: %w", err)
		}
		jsonBlobStr := string(jsonBlob)
		jsonBlobStr = strings.ReplaceAll(jsonBlobStr, "\"initial_height\":1", "\"initial_height\":\"1\"")
		fmt.Println(jsonBlobStr)
		genDoc, err := types.GenesisDocFromJSON([]byte(jsonBlobStr))
		if err != nil {
			return nil, fmt.Errorf("error reading GenesisDoc at %s: %w", cmtConfig.GenesisFile(), err)
		}
		return genDoc, nil
	}
	metricsProvider := func(chainID string) *p2p.Metrics {
		if cmtConfig.Instrumentation.Prometheus {
			return p2p.PrometheusMetrics(cmtConfig.Instrumentation.Namespace, "chain_id", chainID)
		}
		return p2p.NopMetrics()
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(tiasyncCfg.NodeKeyFile(cmtConfig.BaseConfig))
	if err != nil {
		panic(err)
	}

	cometNodeKey, err := p2p.LoadOrGenNodeKey(cmtConfig.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	blockStore, stateDB, err := initDBs(cmtConfig, dbProvider)
	if err != nil {
		return nil, err
	}

	state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)
	if err != nil {
		return nil, err
	}
	p2pMetrics := metricsProvider(genDoc.ChainID)

	bcReactor := blocksync.NewReactor(state, blockStore, cometNodeKey.ID(), celestiaCfg, genDoc, clientCtx, cmtConfig, tiasyncCfg.TiaPollInterval, celestiaNamespace, tiasyncCfg.ChainID)
	bcReactor.SetLogger(logger.With("tsmodule", "tsblocksync"))

	stateSyncReactor := statesync.NewReactor(
		*cmtConfig.StateSync,
		cometNodeKey.ID(),
	)
	stateSyncReactor.SetLogger(logger.With("tsmodule", "tsstatesync"))

	nodeInfo, err := makeNodeInfo(cmtConfig, nodeKey, genDoc, state)
	if err != nil {
		return nil, err
	}

	transport, peerFilters := createTransport(cmtConfig, nodeInfo, nodeKey)

	p2pLogger := logger.With("tsmodule", "tsp2p")

	mempoolReactor := mempool.NewReactor(cmtConfig.Mempool, cometNodeKey.ID())
	mempoolLogger := logger.With("tsmodule", "tsmempool")
	mempoolReactor.SetLogger(mempoolLogger)

	consensusReactor := consensus.NewReactor(cometNodeKey.ID())
	consensusLogger := logger.With("tsmodule", "tsconsensus")
	consensusReactor.SetLogger(consensusLogger)

	sw := createSwitch(
		cmtConfig, transport, p2pMetrics, peerFilters, mempoolReactor, bcReactor,
		stateSyncReactor, consensusReactor, nodeInfo, nodeKey, p2pLogger,
	)

	err = sw.AddPersistentPeers(getPersistentPeers(TiasyncInternalCfg.P2P.PersistentPeers, cometNodeKey, cmtConfig.P2P.ListenAddress))
	if err != nil {
		return nil, fmt.Errorf("could not add peers from persistent_peers field: %w", err)
	}

	err = sw.AddUnconditionalPeerIDs(splitAndTrimEmpty(TiasyncInternalCfg.P2P.UnconditionalPeerIDs, ",", " "))
	if err != nil {
		return nil, fmt.Errorf("could not add peer ids from unconditional_peer_ids field: %w", err)
	}

	addrBook, err := createAddrBookAndSetOnSwitch(cmtConfig, tiasyncCfg, sw, p2pLogger, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("could not create addrbook: %w", err)
	}

	pexReactor := createPEXReactorAndAddToSwitch(addrBook, cmtConfig, sw, logger)

	tiasync := &Tiasync{
		cmtConfig:   cmtConfig,
		celestiaCfg: celestiaCfg,
		tiasyncCfg:  tiasyncCfg,
		genesisDoc:  genDoc,

		transport: transport,
		sw:        sw,
		addrBook:  addrBook,
		nodeInfo:  nodeInfo,
		nodeKey:   nodeKey,

		cometNodeKey: cometNodeKey,

		blockStore:       blockStore,
		bcReactor:        bcReactor,
		mempoolReactor:   mempoolReactor,
		consensusReactor: consensusReactor,
		stateSyncReactor: stateSyncReactor,
		pexReactor:       pexReactor,
		Logger:           logger,
	}

	return tiasync, nil

}

func (t *Tiasync) Start() {
	// Start the transport.
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(t.nodeKey.ID(), TiasyncInternalCfg.P2P.ListenAddress))
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
	err = t.sw.DialPeersAsync(getPersistentPeers(TiasyncInternalCfg.P2P.PersistentPeers, t.cometNodeKey, t.cmtConfig.P2P.ListenAddress))
	if err != nil {
		panic(fmt.Errorf("could not dial peers from persistent_peers field: %w", err))
	}
}
