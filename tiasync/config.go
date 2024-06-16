package tiasync

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

const (
	FlagAddrBookPath    = "tiasync.addr-book-path"
	FlagChainID         = "tiasync.chain-id"
	FlagEnable          = "tiasync.enable"
	FlagListenAddress   = "tiasync.laddr"
	FlagNodeKeyPath     = "tiasync.node-key-path"
	FlagTiaPollInterval = "tiasync.tia-poll-interval"
	FlagUpstreamPeers   = "tiasync.upstream-peers"

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

	# Address to listen for incoming connections from the validator network
	laddr = ""

	# Path to the JSON file containing the private key to use for node authentication in the p2p protocol
	# Tiasync must have a different key than the full node's cometbft instance
	node-key-path = "config/tiasync_node_key.json"

	# Cadence to query celestia for new blocks
	tia-poll-interval = "5s"

	# Peers that are upstream from this node and on the validator network
	upstream-peers = ""
	`
)

var (
	defaultNodeKeyPath  = filepath.Join(DefaultConfigDir, DefaultNodeKeyName)
	defaultAddrBookPath = filepath.Join(DefaultConfigDir, DefaultAddrBookName)
)

var DefaultTiasyncConfig = TiasyncConfig{
	AddrBookPath: defaultAddrBookPath,
	ChainID: "",
	Enable:   false,
	ListenAddress: "",
	NodeKeyPath: defaultNodeKeyPath,
	TiaPollInterval: time.Second * 5,
	UpstreamPeers:      "",
}

// TiasyncConfig defines the configuration for the in-process tiasync.
type TiasyncConfig struct {
	// Path to tiasync's address book
	AddrBookPath string `mapstructure:"addr-book-path"`

	// Optionally provide the chain-id to filter out blocks from our namespace with a different chain-id
	ChainID string `mapstructure:"chain-id"`

	// Switch to enable/disable tiasync
	Enable bool `mapstructure:"enable"`

	// Address to listen for incoming connections from the validator network
	ListenAddress string `mapstructure:"laddr"`

	// Path to the JSON file containing the private key to use for node authentication in the p2p protocol
	// Tiasync must have a different key than the full node's cometbft instance
	NodeKeyPath string `mapstructure:"node-key-path"`

	// Cadence to query celestia for new blocks
	TiaPollInterval time.Duration `mapstructure:"tia-poll-interval"`

	// Peers that are upstream from this node and on the validator network
	UpstreamPeers string `mapstructure:"upstream-peers"`
}

func TiasyncConfigFromAppOpts(appOpts servertypes.AppOptions) TiasyncConfig {
	return TiasyncConfig{
		AddrBookPath:    cast.ToString(appOpts.Get(FlagAddrBookPath)),
		ChainID:         cast.ToString(appOpts.Get(FlagChainID)),
		Enable:          cast.ToBool(appOpts.Get(FlagEnable)),
		ListenAddress:   cast.ToString(appOpts.Get(FlagListenAddress)),
		NodeKeyPath:     cast.ToString(appOpts.Get(FlagNodeKeyPath)),
		TiaPollInterval: cast.ToDuration(appOpts.Get(FlagTiaPollInterval)),
		UpstreamPeers:   cast.ToString(appOpts.Get(FlagUpstreamPeers)),
	}
}

func (t TiasyncConfig) NodeKeyFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.NodeKeyPath, cometCfg.RootDir)
}

func (t TiasyncConfig) AddrBookFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.AddrBookPath, cometCfg.RootDir)
}

// only if tiasync is enabled
func verifyConfigs(tiasyncCfg *TiasyncConfig, cometCfg *cfg.Config) error {
	if tiasyncCfg.Enable {
		if cometCfg.StateSync.Enable && tiasyncCfg.UpstreamPeers == "" {
			return fmt.Errorf("tiasync enabled, must have at least one tiasync upstream peer with state sync enabled")
		}
		if !strings.Contains(cometCfg.P2P.ListenAddress, "127.0.0.1") {
			return fmt.Errorf("tiasync enabled, comet config's P2P ListenAddress/laddr must contain 127.0.0.1")
		}
		if cometCfg.P2P.Seeds != "" || cometCfg.P2P.PersistentPeers != "" {
			return fmt.Errorf("tiasync enabled, comet config's P2P seeds/persistent peers must be empty")
		}
		if cometCfg.P2P.AddrBookStrict {
			return fmt.Errorf("tiasync enabled, comet config's P2P address book strict must be false")
		}
		if !cometCfg.P2P.AllowDuplicateIP {
			return fmt.Errorf("tiasync enabled, comet config's P2P allow duplicate IP must be true")
		}
		if cometCfg.P2P.PexReactor {
			return fmt.Errorf("tiasync enabled, comet config's P2P pex reactor must be false")
		}
		if cometCfg.P2P.ExternalAddress != "" {
			return fmt.Errorf("tiasync enabled, comet config's P2P external address must be empty")
		}
	}
	return nil
}

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}