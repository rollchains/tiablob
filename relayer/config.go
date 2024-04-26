package relayer

import (
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

const (
	FlagAppRpcURL     = "celestia.app-rpc-url"
	FlagAppRpcTimeout = "celestia.app-rpc-timeout"
	FlagChainID       = "celestia.chain-id"
	FlagGasPrices     = "celestia.gas-prices"
	FlagGasAdjustment = "celestia.gas-adjustment"
	FlagNodeRpcURL    = "celestia.node-rpc-url"
	FlagNodeAuthToken = "celestia.node-auth-token"
	FlagQueryInterval = "celestia.proof-query-interval"
	FlagMaxFlushSize  = "celestia.max-flush-size"
)

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

	// Query Celestia for new block proofs this often
	ProofQueryInterval time.Duration `mapstructure:"proof-query-interval"`

	// Only flush at most this many block proofs in an injected tx per block proposal
	MaxFlushSize int `mapstructure:"max-flush-size"`
}

func CelestiaConfigFromAppOpts(appOpts servertypes.AppOptions) CelestiaConfig {
	return CelestiaConfig{
		AppRpcURL:          cast.ToString(appOpts.Get(FlagAppRpcURL)),
		AppRpcTimeout:      cast.ToDuration(appOpts.Get(FlagAppRpcTimeout)),
		ChainID:            cast.ToString(appOpts.Get(FlagChainID)),
		GasPrice:           cast.ToString(appOpts.Get(FlagGasPrices)),
		GasAdjustment:      cast.ToFloat64(appOpts.Get(FlagGasAdjustment)),
		NodeRpcURL:         cast.ToString(appOpts.Get(FlagNodeRpcURL)),
		NodeAuthToken:      cast.ToString(appOpts.Get(FlagNodeAuthToken)),
		ProofQueryInterval: cast.ToDuration(appOpts.Get(FlagQueryInterval)),
		MaxFlushSize:       cast.ToInt(appOpts.Get(FlagMaxFlushSize)),
	}
}
