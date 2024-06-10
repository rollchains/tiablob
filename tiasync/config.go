package tiasync

import (
	"path/filepath"

	cfg "github.com/cometbft/cometbft/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

const (
	FlagAddrBookPath  = "tiasync.addr-book-path"
	FlagEnable        = "tiasync.enable"
	FlagListenAddress = "tiasync.laddr"
	FlagNodeKeyPath   = "tiasync.node-key-path"
	FlagUpstreamPeers = "tiasync.upstream-peers"

	DefaultConfigDir    = "config"
	DefaultNodeKeyName  = "tiasync_node_key.json"
	DefaultAddrBookName = "tiasync_addrbook.json"

	DefaultConfigTemplate = `

	[tiasync]
	addr-book-path = "config/tiasync_addrbook.json"

	enable = false

	laddr = ""

	node-key-path = "config/tiasync_addrbook.json"

	upstream-peers = ""
	`
)

var (
	defaultNodeKeyPath  = filepath.Join(DefaultConfigDir, DefaultNodeKeyName)
	defaultAddrBookPath = filepath.Join(DefaultConfigDir, DefaultAddrBookName)
)

var DefaultTiasyncConfig = TiasyncConfig{
	Enable:   false,
	ListenAddress: "",
	UpstreamPeers:      "",
}

// TiasyncConfig defines the configuration for the in-process tiasync.
type TiasyncConfig struct {
	AddrBookPath string `mapstructure:"addr-book-path"`

	Enable bool `mapstructure:"enable"`

	ListenAddress string `mapstructure:"laddr"`

	NodeKeyPath string `mapstructure:"node-key-path"`

	UpstreamPeers string `mapstructure:"upstream-peers"`
}

func TiasyncConfigFromAppOpts(appOpts servertypes.AppOptions) TiasyncConfig {
	return TiasyncConfig{
		AddrBookPath:  cast.ToString(appOpts.Get(FlagAddrBookPath)),
		Enable:        cast.ToBool(appOpts.Get(FlagEnable)),
		ListenAddress: cast.ToString(appOpts.Get(FlagListenAddress)),
		NodeKeyPath:   cast.ToString(appOpts.Get(FlagNodeKeyPath)),
		UpstreamPeers: cast.ToString(appOpts.Get(FlagUpstreamPeers)),
	}
}

func (t TiasyncConfig) NodeKeyFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.NodeKeyPath, cometCfg.RootDir)
}

func (t TiasyncConfig) AddrBookFile(cometCfg cfg.BaseConfig) string {
	return rootify(t.AddrBookPath, cometCfg.RootDir)
}

// only if tiasync is enabled
func verifyConfigs() {
	// If state sync is enabled, must have upstream-peers in tiasync
	// p2p.laddr must be localhost??
	// one node-key is set, persistent_peers must be empty
	// p2p.addr_book_strict must be false
	// p2p.allow_duplicate_ip must be true
	// p2p.pex must be false
	// p2p.external-addr is empty
}

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}