package relayer

import (
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

const (
	FlagAppRpcURL           = "celestia.app-rpc-url"
	FlagAppRpcTimeout       = "celestia.app-rpc-timeout"
	FlagChainID             = "celestia.chain-id"
	FlagGasPrices           = "celestia.gas-prices"
	FlagGasAdjustment       = "celestia.gas-adjustment"
	FlagNodeRpcURL          = "celestia.node-rpc-url"
	FlagNodeAuthToken       = "celestia.node-auth-token"
	FlagOverrideNamespace   = "celestia.override-namespace"
	FlagOverridePubInterval = "celestia.override-pub-interval"
	FlagQueryInterval       = "celestia.proof-query-interval"
	FlagMaxFlushSize        = "celestia.max-flush-size"

	DefaultConfigTemplate = `

	[celestia]
	# RPC URL of celestia-app node for posting block data, querying proofs & light blocks
	app-rpc-url = "https://rpc-mocha.pops.one:443"

	# RPC Timeout for transaction broadcasts and queries to celestia-app node
	app-rpc-timeout = "30s"

	# Celestia chain id
	chain-id = "celestia-1"

	# Gas price to pay for celestia transactions
	gas-prices = "0.01utia"

	# Gas adjustment for celestia transactions
	gas-adjustment = 1.0

	# RPC URL of celestia-node for querying blobs
	node-rpc-url = "http://127.0.0.1:26658"

	# Auth token for celestia-node RPC, n/a if --rpc.skip-auth is used on start
	node-auth-token = "auth-token"

	# Overrides the expected chain's namespace, test-only
	override-namespace = ""

	# Overrides the expected chain's publish-to-celestia block interval, test-only
	override-pub-interval = 0
	
	# Query celestia for new block proofs this often
	proof-query-interval = "12s"

	# Only flush at most this many block proofs in an injected tx per block proposal
	# Must be greater than 0 and less than 100, proofs are roughly 1KB each
	# tiablob will try to aggregate multiple blobs published at the same height w/ a single proof
	max-flush-size = 32

	`
)

var DefaultCelestiaConfig = CelestiaConfig{
	AppRpcURL:          "https://rpc-mocha.pops.one:443", // TODO remove hardcoded URL
	AppRpcTimeout:      30 * time.Second,
	ChainID:            "celestia-1",
	GasPrice:           "0.01utia",
	GasAdjustment:      1.0,
	NodeRpcURL:         "http://127.0.0.1:26658",
	NodeAuthToken:      "auth-token",
	ProofQueryInterval: 12 * time.Second,
	MaxFlushSize:       32,
}

// CelestiaConfig defines the configuration for the in-process Celestia relayer.
type CelestiaConfig struct {
	// RPC URL of celestia-app
	AppRpcURL string `mapstructure:"app-rpc-url"`

	// RPC Timeout for celestia-app
	AppRpcTimeout time.Duration `mapstructure:"app-rpc-timeout"`

	// Celestia chain ID
	ChainID string `mapstructure:"chain-id"`

	// Gas price to pay for Celestia transactions
	GasPrice string `mapstructure:"gas-prices"`

	// Gas adjustment for Celestia transactions
	GasAdjustment float64 `mapstructure:"gas-adjustment"`

	// RPC URL of celestia-node
	NodeRpcURL string `mapstructure:"node-rpc-url"`

	// RPC Timeout for celestia-node
	NodeAuthToken string `mapstructure:"node-auth-token"`

	// Overrides built-in namespace used
	OverrideNamespace string `mapstructure:"override-namespace"`

	// Overrides built-in publish-to-celestia block interval
	OverridePubInterval int `mapstructure:"override-pub-interval"`

	// Query Celestia for new block proofs this often
	ProofQueryInterval time.Duration `mapstructure:"proof-query-interval"`

	// Only flush at most this many block proofs in an injected tx per block proposal
	MaxFlushSize int `mapstructure:"max-flush-size"`
}

func CelestiaConfigFromAppOpts(appOpts servertypes.AppOptions) CelestiaConfig {
	return CelestiaConfig{
		AppRpcURL:           cast.ToString(appOpts.Get(FlagAppRpcURL)),
		AppRpcTimeout:       cast.ToDuration(appOpts.Get(FlagAppRpcTimeout)),
		ChainID:             cast.ToString(appOpts.Get(FlagChainID)),
		GasPrice:            cast.ToString(appOpts.Get(FlagGasPrices)),
		GasAdjustment:       cast.ToFloat64(appOpts.Get(FlagGasAdjustment)),
		NodeRpcURL:          cast.ToString(appOpts.Get(FlagNodeRpcURL)),
		NodeAuthToken:       cast.ToString(appOpts.Get(FlagNodeAuthToken)),
		OverrideNamespace:   cast.ToString(appOpts.Get(FlagOverrideNamespace)),
		OverridePubInterval: cast.ToInt(appOpts.Get(FlagOverridePubInterval)),
		ProofQueryInterval:  cast.ToDuration(appOpts.Get(FlagQueryInterval)),
		MaxFlushSize:        cast.ToInt(appOpts.Get(FlagMaxFlushSize)),
	}
}

var RelayerInternalCfg RelayerInternalConfig
type RelayerInternalConfig struct {
	// Database backend: goleveldb | cleveldb | boltdb | rocksdb
	// * goleveldb (github.com/syndtr/goleveldb - most popular implementation)
	//   - pure go
	//   - stable
	// * cleveldb (uses levigo wrapper)
	//   - fast
	//   - requires gcc
	//   - use cleveldb build tag (go build -tags cleveldb)
	// * boltdb (uses etcd's fork of bolt - github.com/etcd-io/bbolt)
	//   - EXPERIMENTAL
	//   - may be faster is some use-cases (random reads - indexer)
	//   - use boltdb build tag (go build -tags boltdb)
	// * rocksdb (uses github.com/tecbot/gorocksdb)
	//   - EXPERIMENTAL
	//   - requires gcc
	//   - use rocksdb build tag (go build -tags rocksdb)
	// * badgerdb (uses github.com/dgraph-io/badger)
	//   - EXPERIMENTAL
	//   - use badgerdb build tag (go build -tags badgerdb)
	DBBackend string

	// Database directory
	DBPath string
}