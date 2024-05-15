package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestResubmission sets up celestia and a rollchain chains.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// go test -timeout 15m -v -run TestResubmission1 . -count 1
func TestResubmission1(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// TestResubmission2 sets up celestia and 2 rollchain chains, each with a different namespace, both posting to Celestia.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// Repeats with 2 minute pause (longer than expiration)
// go test -timeout 15m -v -run TestResubmission2 . -count 1
func TestResubmission2(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 2)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
}

// TestResubmission3 sets up celestia and 3 rollchain chains, each with a different namespace, all posting to Celestia.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// Repeats with 2 minute pause (longer than expiration) and another 1 min pause
// go test -timeout 20m -v -run TestResubmission3 . -count 1
func TestResubmission3(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 3)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// TestResubmission4 sets up celestia and 4 rollchain chains, each with a different namespace, all posting to Celestia.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// Repeats with 2 minute pause (longer than expiration) and another (2) 1 min pauses
// go test -timeout 20m -v -run TestResubmission4 . -count 1
func TestResubmission4(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 4)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// TestResubmission9 sets up celestia and 9 rollchain chains, each with a different namespace, all posting to Celestia.
// Proves 20 blocks, pauses Celestia for 1 minute and resumes, recovering blocks that weren't posted when Celestia was down.
// Repeats with 2 minute pause (longer than expiration) and another (2) 1 min pauses
// go test -timeout 20m -v -run TestResubmission9 . -count 1
func TestResubmission9(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 9)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}

// TestResubmissionHour is an hour long running test.
// It sets up celestia and 3 rollchain chains, each with a different namespace, all posting to Celestia.
// It pauses Celestia a number of times for different lengths
// go test -timeout 60m -v -run TestResubmissionHour . -count 1
func TestResubmissionHour(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 3)

	m := NewMetrics()
	defer m.PrintMetrics(t)

	m.proveXBlocks(t, ctx, chains.RollchainChain, 20)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 2*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 10*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, 5*time.Minute)
	m.pauseCelestiaAndRecover(t, ctx, chains.RollchainChain, chains.CelestiaChain, time.Minute)
}
