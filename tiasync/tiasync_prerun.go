package tiasync

import (
	"fmt"
	"os"
	"strings"

	"github.com/rollchains/tiablob/tiasync/extcommit"
	"github.com/spf13/cobra"

	dbm "github.com/cometbft/cometbft-db"
	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	"github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/version"
	"github.com/cosmos/cosmos-sdk/server"
)

type TiasyncPrerun struct {

	// config
	cmtConfig  *cfg.Config
	tiasyncCfg *TiasyncConfig
	genesisDoc *types.GenesisDoc // initial validator set

	// network
	transport   *p2p.MultiplexTransport
	sw          *p2p.Switch  // p2p connections
	addrBook    pex.AddrBook // known peers
	nodeInfo    p2p.NodeInfo
	nodeKey     *p2p.NodeKey // our node privkey
	isListening bool

	// services
	extCommitReactor p2p.Reactor  // for fetching extended commit
	pexReactor       *pex.Reactor // for exchanging peer addresses

	Logger log.Logger
}

func TiasyncPrerunRoutine(appName string, cmd *cobra.Command, srvCtx *server.Context) error {
	cometCfg := srvCtx.Config
	tiasyncCfg := TiasyncConfigFromAppOpts(srvCtx.Viper)
	copyConfig(cometCfg)

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	logger, err := cmtflags.ParseLogLevel(cometCfg.LogLevel, logger, cfg.DefaultLogLevel)
	if err != nil {
		panic(err)
	}

	// only run on start
	if cmd.Use != "start" || cmd.Parent().Use != appName {
		return nil
	}

	if !tiasyncCfg.Enable {
		return nil
	}

	if err := verifyAndModifyConfigs(&tiasyncCfg, cometCfg); err != nil {
		panic(err)
	}

	logger.Info("Tiasync enabled")
	dbProvider := cfg.DefaultDBProvider
	genesisDocProvider := func() (*types.GenesisDoc, error) {
		jsonBlob, err := os.ReadFile(cometCfg.GenesisFile())
		if err != nil {
			return nil, fmt.Errorf("couldn't read GenesisDoc file: %w", err)
		}
		jsonBlobStr := string(jsonBlob)
		jsonBlobStr = strings.ReplaceAll(jsonBlobStr, "\"initial_height\":1", "\"initial_height\":\"1\"")
		fmt.Println(jsonBlobStr)
		genDoc, err := types.GenesisDocFromJSON([]byte(jsonBlobStr))
		if err != nil {
			return nil, fmt.Errorf("error reading GenesisDoc at %s: %w", cometCfg.GenesisFile(), err)
		}
		return genDoc, nil
	}

	// Unlike tiasync, tiasync-prerun uses primary block and state DBs, if replay is required, a restart may also be required...
	var blockStoreDB dbm.DB
	blockStoreDB, err = dbProvider(&cfg.DBContext{ID: "blockstore", Config: cometCfg})
	if err != nil {
		return err
	}
	defer blockStoreDB.Close()

	var stateDB dbm.DB
	stateDB, err = dbProvider(&cfg.DBContext{ID: "state", Config: cometCfg})
	if err != nil {
		return err
	}
	defer stateDB.Close()

	var tsstateDB dbm.DB
	tsstateDB, err = dbProvider(&cfg.DBContext{ID: "tsstate", Config: cometCfg})
	if err != nil {
		return err
	}
	defer tsstateDB.Close()

	state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)
	if err != nil {
		return err
	}

	extCommitNeeded, height := isExtCommitNeeded(logger, blockStoreDB, state)
	if !extCommitNeeded {
		return nil
	}

	if TiasyncInternalCfg.P2P.PersistentPeers == "" {
		return fmt.Errorf("vote extensions enabled, persistent peer needed for extented commit")
	}

	err = copyState(state, tsstateDB)
	if err != nil {
		return err
	}

	ts, err := NewTiasyncPrerun(cometCfg, &tiasyncCfg, logger, blockStoreDB, state, genDoc, height)
	if err != nil {
		panic(err)
	}

	// Nothing to do/start
	if ts.extCommitReactor == nil {
		return nil
	}

	ts.Start()

	for {
		select {
		case <-ts.extCommitReactor.Quit():
			_ = ts.sw.Stop()
			ts.transport.Close()
			return nil
		}
	}
}

func NewTiasyncPrerun(
	cmtConfig *cfg.Config,
	tiasyncCfg *TiasyncConfig,
	logger log.Logger,
	blockStoreDB dbm.DB,
	state sm.State,
	genDoc *types.GenesisDoc,
	height int64,
) (*TiasyncPrerun, error) {
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

	p2pMetrics := metricsProvider(genDoc.ChainID)

	extCommitReactor := extcommit.NewReactor(state, logger, blockStoreDB, height)

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
			extcommit.BlocksyncChannel,
			pex.PexChannel,
		},
		Moniker: "tiasync_prerun",
		Other: p2p.DefaultNodeInfoOther{
			TxIndex:    "off",
			RPCAddress: "tcp://0.0.0.0:26657", // not used? but is validated...
		},
		ListenAddr: TiasyncInternalCfg.P2P.ListenAddress,
	}

	transport, peerFilters := createTransport(cmtConfig, nodeInfo, nodeKey)

	p2pLogger := logger.With("tsmodule", "tsp2p")

	sw := p2p.NewSwitch(cmtConfig.P2P, transport, p2p.WithMetrics(p2pMetrics), p2p.SwitchPeerFilters(peerFilters...))
	sw.SetLogger(p2pLogger)
	sw.AddReactor("BLOCKSYNC", extCommitReactor)
	sw.SetNodeInfo(nodeInfo)
	sw.SetNodeKey(nodeKey)
	p2pLogger.Info("P2P Node ID", "ID", nodeKey.ID())

	// TODO: add support for seeds (after intercepting comet config)
	err = sw.AddPersistentPeers(splitAndTrimEmpty(TiasyncInternalCfg.P2P.PersistentPeers, ",", " "))
	if err != nil {
		return nil, fmt.Errorf("could not add peers from persistent_peers field: %w", err)
	}
	addrBook, err := createAddrBookAndSetOnSwitch(cmtConfig, tiasyncCfg, sw, p2pLogger, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("could not create addrbook: %w", err)
	}

	pexReactor := createPEXReactorAndAddToSwitch(addrBook, cmtConfig, sw, logger)

	return &TiasyncPrerun{
		cmtConfig:  cmtConfig,
		tiasyncCfg: tiasyncCfg,
		genesisDoc: genDoc,

		transport: transport,
		sw:        sw,
		addrBook:  addrBook,
		nodeInfo:  nodeInfo,
		nodeKey:   nodeKey,

		extCommitReactor: extCommitReactor,
		pexReactor:       pexReactor,

		Logger: logger,
	}, nil
}

func (t *TiasyncPrerun) Start() {
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
	err = t.sw.DialPeersAsync(splitAndTrimEmpty(TiasyncInternalCfg.P2P.PersistentPeers, ",", " "))
	if err != nil {
		panic(fmt.Errorf("could not dial peers from persistent_peers field: %w", err))
	}
}

func isExtCommitNeeded(
	logger log.Logger,
	blockStoreDB dbm.DB,
	state sm.State,
) (bool, int64) {
	blockStore := store.NewBlockStore(blockStoreDB)
	blockStoreHeight := blockStore.Height()
	if blockStoreHeight < 1 {
		return false, 0
	}

	if !state.ConsensusParams.ABCI.VoteExtensionsEnabled(blockStoreHeight) {
		return false, 0
	}

	// Height is > 0 and Vote extensions are enabled
	extendedCommit := blockStore.LoadBlockExtendedCommit(blockStoreHeight)
	if extendedCommit == nil {
		return true, blockStoreHeight
	}

	voteSet := types.NewExtendedVoteSet(state.ChainID, extendedCommit.Height, extendedCommit.Round, cmtproto.PrecommitType, state.LastValidators)
	for idx, ecs := range extendedCommit.ExtendedSignatures {
		if ecs.BlockIDFlag == types.BlockIDFlagAbsent {
			continue // OK, some precommits can be missing.
		}
		vote := extendedCommit.GetExtendedVote(int32(idx))
		if err := vote.ValidateBasic(); err != nil {
			logger.Info("Lastest heights extended commit failed validate basic", "error", err)
			return true, blockStoreHeight
		}
		added, err := voteSet.AddVote(vote)
		if !added || err != nil {
			logger.Info("Lastest height's extended commit failed to reconstruct vote set", "error", err)
			return true, blockStoreHeight
		}
	}

	return false, 0
}

func copyState(state sm.State, tsstateDB dbm.DB) error {
	tsstateStore := sm.NewStore(tsstateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})

	return tsstateStore.Save(state)
}
