package tiasync

import (
	"fmt"
	"path/filepath"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"

	"github.com/rollchains/tiablob/relayer"
)

const (
	FlagAddrBookPath    = "tiasync.addr-book-path"
	FlagChainID         = "tiasync.chain-id"
	FlagEnable          = "tiasync.enable"
	FlagLocalhostPort   = "tiasync.localhost-port"
	FlagNodeKeyPath     = "tiasync.node-key-path"
	FlagTiaPollInterval = "tiasync.tia-poll-interval"

	DefaultConfigDir    = "config"
	DefaultNodeKeyName  = "tiasync_node_key.json"
	DefaultAddrBookName = "tiasync_addrbook.json"

	DefaultConfigTemplate = `

	[tiasync]
	# Path to tiasync's address book
	addr-book-path = "config/tiasync_addrbook.json"

	# Optionally provide the chain-id to filter out blocks from our namespace with a different chain-id
	chain-id = ""

	# Switch to enable/disable tiasync
	enable = false

	# Port that cometbft will listen on
	localhost-port = "26777"

	# Path to the JSON file containing the private key to use for node authentication in the p2p protocol
	# Tiasync must have a different key than the full node's cometbft instance
	node-key-path = "config/tiasync_node_key.json"

	# Cadence to query celestia for new blocks
	tia-poll-interval = "5s"
	`
)

var (
	defaultNodeKeyPath  = filepath.Join(DefaultConfigDir, DefaultNodeKeyName)
	defaultAddrBookPath = filepath.Join(DefaultConfigDir, DefaultAddrBookName)
)

var DefaultTiasyncConfig = TiasyncConfig{
	AddrBookPath:    defaultAddrBookPath,
	ChainID:         "",
	Enable:          false,
	LocalhostPort:   "26777",
	NodeKeyPath:     defaultNodeKeyPath,
	TiaPollInterval: time.Second * 5,
}

// TiasyncConfig defines the configuration for the in-process tiasync.
type TiasyncConfig struct {
	// Path to tiasync's address book
	AddrBookPath string `mapstructure:"addr-book-path"`

	// Optionally provide the chain-id to filter out blocks from our namespace with a different chain-id
	ChainID string `mapstructure:"chain-id"`

	// Switch to enable/disable tiasync
	Enable bool `mapstructure:"enable"`

	// Port that cometbft will listen on
	LocalhostPort string `mapstructure:"localhost-port"`

	// Path to the JSON file containing the private key to use for node authentication in the p2p protocol
	// Tiasync must have a different key than the full node's cometbft instance
	NodeKeyPath string `mapstructure:"node-key-path"`

	// Cadence to query celestia for new blocks
	TiaPollInterval time.Duration `mapstructure:"tia-poll-interval"`
}

var TiasyncInternalCfg TiasyncInternalConfig

type TiasyncInternalConfig struct {
	P2P cfg.P2PConfig
}

func TiasyncConfigFromAppOpts(appOpts servertypes.AppOptions) TiasyncConfig {
	return TiasyncConfig{
		AddrBookPath:    cast.ToString(appOpts.Get(FlagAddrBookPath)),
		ChainID:         cast.ToString(appOpts.Get(FlagChainID)),
		Enable:          cast.ToBool(appOpts.Get(FlagEnable)),
		LocalhostPort:   cast.ToString(appOpts.Get(FlagLocalhostPort)),
		NodeKeyPath:     cast.ToString(appOpts.Get(FlagNodeKeyPath)),
		TiaPollInterval: cast.ToDuration(appOpts.Get(FlagTiaPollInterval)),
	}
}

func (t TiasyncConfig) NodeKeyFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.NodeKeyPath, cometCfg.RootDir)
}

func (t TiasyncConfig) AddrBookFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.AddrBookPath, cometCfg.RootDir)
}

func copyConfig(cometCfg *cfg.Config) {
	relayer.RelayerInternalCfg.DBBackend = cometCfg.DBBackend
	relayer.RelayerInternalCfg.DBPath = cometCfg.DBPath
	TiasyncInternalCfg.P2P = *cometCfg.P2P
}

func verifyAndModifyConfigs(tiasyncCfg *TiasyncConfig, cometCfg *cfg.Config) error {
	if cometCfg.StateSync.Enable && cometCfg.P2P.PersistentPeers == "" {
		return fmt.Errorf("tiasync enabled, must have at least one persistent peer with state sync enabled")
	}

	cometCfg.P2P.AddrBookStrict = false
	cometCfg.P2P.AllowDuplicateIP = true
	cometCfg.P2P.ExternalAddress = ""
	cometCfg.P2P.ListenAddress = "tcp://127.0.0.1:" + tiasyncCfg.LocalhostPort
	cometCfg.P2P.PersistentPeers = ""
	cometCfg.P2P.PexReactor = false
	cometCfg.P2P.Seeds = ""

	return nil
}

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
