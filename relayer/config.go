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
)

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
