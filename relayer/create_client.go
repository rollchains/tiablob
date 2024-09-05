package relayer

import (
	"fmt"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	celestia "github.com/rollchains/celestia-da-light-client"
)

var (
	defaultTrustLevel      = celestia.Fraction{Numerator: 2, Denominator: 3}
	defaultTrustingPeriod  = time.Hour * 2
	defaultUnbondingPeriod = time.Hour * 3
	defaultMaxClockDrift   = time.Second * 10
)

// TODO: do create client async, worst case scenario, it will be created in the same block as the first proofs
func (r *Relayer) CreateClient(ctx sdk.Context) *celestia.CreateClient {
	latestHeight, err := r.celestiaProvider.QueryLatestHeight(ctx)
	if err != nil {
		r.logger.Error("error querying latest height for create client")
		return nil
	}

	lightBlock, err := r.celestiaProvider.QueryLightBlock(ctx, latestHeight)
	if err != nil {
		r.logger.Error("error querying light block for create client", "error", err)
		return nil
	}

	return &celestia.CreateClient{
		ClientState: *celestia.NewClientState(
			r.celestiaChainID,      // chainID
			defaultTrustLevel,      // trustLevel (TODO: make this configurable?)
			defaultTrustingPeriod,  // trusting period (TODO: set to 2/3 unbonding period, add override?)
			defaultUnbondingPeriod, // unbonding period (TODO: query unbonding period, add override?)
			defaultMaxClockDrift,   // max clock drift (TODO: make this configurable?)
			celestia.Height{RevisionNumber: celestia.ParseChainID(r.celestiaChainID), RevisionHeight: uint64(latestHeight)}, // latest height
		),
		ConsensusState: *celestia.NewConsensusState(
			lightBlock.SignedHeader.Time,               // timestamp
			lightBlock.SignedHeader.DataHash.Bytes(),   // root
			lightBlock.SignedHeader.NextValidatorsHash, // next val hash
		),
	}
}

func (r *Relayer) ValidateNewClient(ctx sdk.Context, client *celestia.CreateClient) error {
	latestHeight, err := r.celestiaProvider.QueryLatestHeight(ctx)
	if err != nil {
		return fmt.Errorf("error querying latest height for create client, %v", err)
	}

	lightBlock, err := r.celestiaProvider.QueryLightBlock(ctx, int64(client.ClientState.LatestHeight.RevisionHeight))
	if err != nil {
		return fmt.Errorf("error querying light block for verify new client, %v", err)
	}

	if client.ClientState.ChainId != r.celestiaChainID {
		return fmt.Errorf("verify new client: invalid chain id")
	}

	if client.ClientState.TrustLevel.Numerator != defaultTrustLevel.Numerator ||
		client.ClientState.TrustLevel.Denominator != defaultTrustLevel.Denominator {
		return fmt.Errorf("verify new client: invalid trust level")
	}

	if client.ClientState.TrustingPeriod != defaultTrustingPeriod {
		return fmt.Errorf("verify new client: invalid trusting period")
	}

	if client.ClientState.UnbondingPeriod != defaultUnbondingPeriod {
		return fmt.Errorf("verify new client: invalid unbonding period")
	}

	if client.ClientState.LatestHeight.RevisionNumber != celestia.ParseChainID(r.celestiaChainID) {
		return fmt.Errorf("verify new client: invalid revision number")
	}

	// New clients must be within 100 blocks of latest height
	if (uint64(latestHeight) - client.ClientState.LatestHeight.RevisionHeight) > 100 {
		return fmt.Errorf("verify new client: invalid revision height")
	}

	if !reflect.DeepEqual(client.ConsensusState.Root, lightBlock.SignedHeader.DataHash.Bytes()) {
		return fmt.Errorf("verify new client: invalid root hash")
	}

	if client.ConsensusState.Timestamp != lightBlock.SignedHeader.Time {
		return fmt.Errorf("verify new client: invalid timestamp")
	}

	if !reflect.DeepEqual(client.ConsensusState.NextValidatorsHash, lightBlock.SignedHeader.NextValidatorsHash) {
		return fmt.Errorf("verify new client: invalid next validator hash")
	}

	return nil
}
