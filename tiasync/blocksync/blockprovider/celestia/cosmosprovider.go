package celestia

import (
	"context"
	"time"

	celestiarpc "github.com/tendermint/tendermint/rpc/client/http"
)

type CosmosProvider struct {
	rpcClient *celestiarpc.HTTP
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider(rpcURL string, timeout time.Duration) (*CosmosProvider, error) {
	rpcClient, err := celestiarpc.NewWithTimeout(rpcURL, "/websocket", uint(timeout.Seconds()))
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		rpcClient: rpcClient,
	}

	return cp, nil
}

// Get that starting celestia height to query
func (cp *CosmosProvider) GetStartingCelestiaHeight(ctx context.Context, genTime time.Time) (int64, error) {
	latestCelestiaHeight, err := cp.QueryLatestHeight(ctx)
	if err != nil {
		return 0, err
	}

	estimatedCelestiaHeight, timeAtEstimatedCelestiaHeight, err := cp.getEstimatedCelestiaHeight(ctx, genTime, latestCelestiaHeight)
	if err != nil {
		return 0, err
	}

	// If genTime is later than estimated celestia height time, continue narrowing (adds blocks)
	// If genTime is >10m than estimated celestia height time, continue narrowing (subtracts blocks)
	estimatedCount := 0
	estimatedAt1Count := 0
	secondsFromFirstPossibleBlock := float64(600) // try to get within 10 minutes of the necessary celestia block (increases as needed)
	for genTime.Sub(timeAtEstimatedCelestiaHeight).Seconds() < 0 || genTime.Sub(timeAtEstimatedCelestiaHeight).Seconds() > secondsFromFirstPossibleBlock {
		estimatedCelestiaHeight, timeAtEstimatedCelestiaHeight, err = cp.getEstimatedCelestiaHeight(ctx, genTime, estimatedCelestiaHeight)
		if err != nil {
			return 0, err
		}

		estimatedCount++
		if estimatedCelestiaHeight == 1 {
			estimatedAt1Count++
		}

		// This should only occur during testing and/or new testnets
		if estimatedAt1Count > 3 {
			break
		}

		// After 5 calculations, increase our threshold (could be due to a celestia halt)
		if estimatedCount > 5 {
			secondsFromFirstPossibleBlock = secondsFromFirstPossibleBlock * 2
		}
	}

	return estimatedCelestiaHeight, nil
}

func (cp *CosmosProvider) getEstimatedCelestiaHeight(ctx context.Context, genTime time.Time, celestiaHeight int64) (int64, time.Time, error) {
	timeAtCelestiaHeight, err := cp.QueryTimeAtHeight(ctx, &celestiaHeight)
	if err != nil {
		return 0, time.Time{}, err
	}

	celestiaHeightm10 := celestiaHeight - 10
	timeAtCelestiaHeightm10, err := cp.QueryTimeAtHeight(ctx, &celestiaHeightm10)
	if err != nil {
		return 0, time.Time{}, err
	}

	celestiaBlockTimeMs := timeAtCelestiaHeight.Sub(timeAtCelestiaHeightm10).Milliseconds() / 10
	estimatedCelestiaHeight := celestiaHeight - timeAtCelestiaHeight.Sub(genTime).Milliseconds()/celestiaBlockTimeMs - 1

	if estimatedCelestiaHeight <= 0 {
		estimatedCelestiaHeight = 1
	}

	timeAtEstimatedCelestiaHeight, err := cp.QueryTimeAtHeight(ctx, &estimatedCelestiaHeight)
	if err != nil {
		return 0, time.Time{}, err
	}

	return estimatedCelestiaHeight, timeAtEstimatedCelestiaHeight, nil
}
