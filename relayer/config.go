package relayer

import (
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

const (
	FlagRpcURL        = "celestia.rpc-url"
	FlagRpcTimeout    = "celestia.rpc-timeout"
	FlagGasPrices     = "celestia.gas-prices"
	FlagGasAdjustment = "celestia.gas-adjustment"
	FlagQueryInterval = "celestia.proof-query-interval"
	FlagMaxFlushSize  = "celestia.max-flush-size"

	DefaultConfigTemplate = `

	[celestia]
	# RPC URL of celestia-app node for posting block data
	# TODO remove hardcoded URL
	rpc-url = "https://rpc-mocha.pops.one:443"

	# RPC Timeout for transaction broadcasts and queries to celestia-app node
	rpc-timeout = "30s"

	# Gas price to pay for celestia transactions
	gas-prices = "0.01utia"

	# Gas adjustment for celestia transactions
	gas-adjustment = 1.0

	# Query celestia for new block proofs this often
	proof-query-interval = "12s"

	# Only flush at most this many block proofs in an injected tx per block proposal
	max-flush-size = 32
	`
)

var DefaultCelestiaConfig = CelestiaConfig{
	RpcURL:             "https://rpc-mocha.pops.one:443", // TODO remove hardcoded URL
	RpcTimeout:         30 * time.Second,
	GasPrice:           "0.01utia",
	GasAdjustment:      1.0,
	ProofQueryInterval: 12 * time.Second,
	MaxFlushSize:       32,
}

// CelestiaConfig defines the configuration for the in-process Celestia relayer.
type CelestiaConfig struct {
	// RPC URL of Celestia
	RpcURL string `mapstructure:"rpc-url"`

	// RPC Timeout
	RpcTimeout time.Duration `mapstructure:"rpc-timeout"`

	// Gas price to pay for Celestia transactions
	GasPrice string `mapstructure:"gas-prices"`

	// Gas adjustment for Celestia transactions
	GasAdjustment float64 `mapstructure:"gas-adjustment"`

	// Query Celestia for new block proofs this often
	ProofQueryInterval time.Duration `mapstructure:"proof-query-interval"`

	// Only flush at most this many block proofs in an injected tx per block proposal
	MaxFlushSize int `mapstructure:"max-flush-size"`
}

func CelestiaConfigFromAppOpts(appOpts servertypes.AppOptions) CelestiaConfig {
	return CelestiaConfig{
		RpcURL:             cast.ToString(appOpts.Get(FlagRpcURL)),
		RpcTimeout:         cast.ToDuration(appOpts.Get(FlagRpcTimeout)),
		GasPrice:           cast.ToString(appOpts.Get(FlagGasPrices)),
		GasAdjustment:      cast.ToFloat64(appOpts.Get(FlagGasAdjustment)),
		ProofQueryInterval: cast.ToDuration(appOpts.Get(FlagQueryInterval)),
		MaxFlushSize:       cast.ToInt(appOpts.Get(FlagMaxFlushSize)),
	}
}
