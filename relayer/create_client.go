package relayer


import (
	"fmt"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rollchains/tiablob/light-clients/celestia"
)

var (
	defaultTrustLevel = celestia.Fraction{Numerator: 2, Denominator: 3}
	defaultTrustingPeriod = time.Hour * 2
	defaultUnbondingPeriod = time.Hour * 3
	defaultMaxClockDrift = time.Second * 10
)

// TODO: do create client async, worst case scenario, it will be created in the same block as the first proofs
func (r *Relayer) CreateClient(ctx sdk.Context) *CreateClient {
	latestHeight, err := r.provider.QueryLatestHeight(ctx)
	if err != nil {
		fmt.Println("error querying latest height for create client")
		return nil
	}

	lightBlock, err := r.provider.QueryLightBlock(ctx, latestHeight)
	if err != nil {
		fmt.Println("error querying light block for create client, ", err)
		return nil
	}

	return &CreateClient{
		ClientState: *celestia.NewClientState(
			r.celestiaChainID, // chainID
			defaultTrustLevel, // trustLevel (TODO: make this configurable?)
			defaultTrustingPeriod, // trusting period (TODO: set to 2/3 unbonding period, add override?)
			defaultUnbondingPeriod, // unbonding period (TODO: query unbonding period, add override?)
			defaultMaxClockDrift, // max clock drift (TODO: make this configurable?)
			celestia.Height{RevisionNumber: celestia.ParseChainID(r.celestiaChainID), RevisionHeight: uint64(latestHeight)}, // latest height
		),
		ConsensusState: *celestia.NewConsensusState(
			lightBlock.SignedHeader.Time, // timestamp
			lightBlock.SignedHeader.DataHash.Bytes(), // root
			lightBlock.SignedHeader.NextValidatorsHash, // next val hash
		),
	}
}

func (r *Relayer) ValidateNewClient(ctx sdk.Context, client *CreateClient) bool {
	latestHeight, err := r.provider.QueryLatestHeight(ctx)
	if err != nil {
		fmt.Println("error querying latest height for create client")
		return false
	}

	lightBlock, err := r.provider.QueryLightBlock(ctx, int64(client.ClientState.LatestHeight.RevisionHeight))
	if err != nil {
		fmt.Println("error querying light block for verify new client, ", err)
		return false
	}

	if client.ClientState.ChainId != r.celestiaChainID {
		fmt.Println("verify new client: invalid chain id")
		return false
	}

	if client.ClientState.TrustLevel.Numerator != defaultTrustLevel.Numerator || 
		client.ClientState.TrustLevel.Denominator != defaultTrustLevel.Denominator {
		fmt.Println("verify new client: invalid trust level")
		return false
	}

	if client.ClientState.TrustingPeriod != defaultTrustingPeriod {
		fmt.Println("verify new client: invalid trusting period")
		return false
	}

	if client.ClientState.UnbondingPeriod != defaultUnbondingPeriod {
		fmt.Println("verify new client: invalid unbonding period")
		return false
	}

	if client.ClientState.LatestHeight.RevisionNumber != celestia.ParseChainID(r.celestiaChainID) {
		fmt.Println("verify new client: invalid revision number")
		return false
	}

	// New clients must be within 100 blocks of latest height
	if (uint64(latestHeight) - client.ClientState.LatestHeight.RevisionHeight) > 100 {
		fmt.Println("verify new client: invalid revision height")
		return false
	}

	if !reflect.DeepEqual(client.ConsensusState.Root, lightBlock.SignedHeader.DataHash.Bytes()) {
		fmt.Println("verify new client: invalid root hash")
		return false
	}

	if client.ConsensusState.Timestamp != lightBlock.SignedHeader.Time {
		fmt.Println("verify new client: invalid timestamp")
		return false
	}

	if !reflect.DeepEqual(client.ConsensusState.NextValidatorsHash, lightBlock.SignedHeader.NextValidatorsHash) {
		fmt.Println("verify new client: invalid next validator hash")
		return false
	}

	return true
}