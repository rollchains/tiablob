package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	"github.com/rollchains/rollchains/interchaintest/api/rollchain"
	"github.com/rollchains/rollchains/interchaintest/setup"
)

// TestUpgradeInPlaceMigration performs an upgrade without a genesis restart
// Pre-reqs: build local2 verison of rollchains with version in makefile set to 2
//
//	heighliner build -c rollchain --local -f chains.yaml --go-version 1.22.1 -g local2
//
// go test -timeout 20m -v -run TestUpgradeInPlaceMigration . -count 1
func TestUpgradeInPlaceMigration(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "wallet", userFunds, chains.RollchainChain)
	user := users[0]

	height, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	haltHeightDelta := int64(10)
	haltHeight := height + haltHeightDelta

	upgradeMsg := &upgradetypes.MsgSoftwareUpgrade{
		Authority: types.MustBech32ifyAddressBytes(chains.RollchainChain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Plan: upgradetypes.Plan{
			Name:   "2",
			Height: haltHeight,
			Info:   "",
		},
	}
	prop, err := chains.RollchainChain.BuildProposal(
		[]cosmos.ProtoMessage{upgradeMsg},
		"Upgrade proposal 1",  // title
		"first chain upgrade", // summary
		"[]",                  // metadata
		"500000000"+chains.RollchainChain.Config().Denom, // deposit
		user.FormattedAddress(),                          // proposer
		false,                                            // expedited
	)
	require.NoError(t, err)

	upgradeTx, err := chains.RollchainChain.SubmitProposal(ctx, user.KeyName(), prop)
	require.NoError(t, err)

	err = chains.RollchainChain.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	propId, err := strconv.ParseUint(upgradeTx.ProposalID, 10, 64)
	require.NoError(t, err, "failed to convert proposal ID to uint64")

	_, err = cosmos.PollForProposalStatus(ctx, chains.RollchainChain, height, height+haltHeightDelta, propId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chains.RollchainChain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chains.RollchainChain)

	height, err = chains.RollchainChain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = chains.RollchainChain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	chains.RollchainChain.UpgradeVersion(ctx, chains.Client, "rollchain", "local2")

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chains.RollchainChain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*75)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 20, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

}
// TestUpgradeHandler performs an upgrade from a chain without tiablob to a chian with tiablob, without a genesis restart
// Pre-reqs: 
// * build verison of rollchains without tiablob (branch: steve/tiablobless-rcd)
// * * heighliner build -c rollchain --local -f chains.yaml --go-version 1.22.1 -g tiablobless
// * build local version of rollchain with version set to 2 in makefile
// go test -timeout 20m -v -run TestUpgradeHandler . -count 1
func TestUpgradeHandler(t *testing.T) {
	ctx := context.Background()

	// Get the rollchain spec before starting chains so we can specify the docker version without tiablob,
	// and we can add the default celestia config + our custom values to app.toml (tiablob default config won't exist otherwise)
	rollchainSpec := setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0)
	rollchainSpec.Version = "tiablobless"
	rollchainSpec.Images[0].Version = rollchainSpec.Version
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	rollchainSpec.ConfigFileOverrides = testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":           fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"app-rpc-timeout":       "30s",
				"chain-id":              celestiaChainID,
				"gas-prices":            "0.01utia",
				"gas-adjustment":        1.0,
				"node-rpc-url":          fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"node-auth-token":       "auth-token",
				"override-namespace":    "rc_demo",
				"override-pub-interval": 0,
				"proof-query-interval":  "12s",
				"max-flush-size":        32,
			},
			"state-sync": testutil.Toml{
				"snapshot-interval":    100,
				"snapshot-keep-recent": 2,
			},
		},
	}
	celestiaChainSpec := setup.CelestiaChainSpec()
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec, []*interchaintest.ChainSpec{rollchainSpec})

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "wallet", userFunds, chains.RollchainChain)
	user := users[0]

	height, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	haltHeightDelta := int64(10)
	haltHeight := height + haltHeightDelta

	upgradeMsg := &upgradetypes.MsgSoftwareUpgrade{
		Authority: types.MustBech32ifyAddressBytes(chains.RollchainChain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Plan: upgradetypes.Plan{
			Name:   "2",
			Height: haltHeight,
			Info:   "",
		},
	}
	prop, err := chains.RollchainChain.BuildProposal(
		[]cosmos.ProtoMessage{upgradeMsg},
		"Upgrade proposal 1",  // title
		"first chain upgrade", // summary
		"[]",                  // metadata
		"500000000"+chains.RollchainChain.Config().Denom, // deposit
		user.FormattedAddress(),                          // proposer
		false,                                            // expedited
	)
	require.NoError(t, err)

	upgradeTx, err := chains.RollchainChain.SubmitProposal(ctx, user.KeyName(), prop)
	require.NoError(t, err)

	err = chains.RollchainChain.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	propId, err := strconv.ParseUint(upgradeTx.ProposalID, 10, 64)
	require.NoError(t, err, "failed to convert proposal ID to uint64")

	_, err = cosmos.PollForProposalStatus(ctx, chains.RollchainChain, height, height+haltHeightDelta, propId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chains.RollchainChain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chains.RollchainChain)

	height, err = chains.RollchainChain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = chains.RollchainChain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	chains.RollchainChain.UpgradeVersion(ctx, chains.Client, "rollchain", "local")

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chains.RollchainChain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*3)
	defer timeoutCtxCancel()
	
	// wait 1 block to be sure chain has started
	err = testutil.WaitForBlocks(timeoutCtx, 1, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	origProvenHeight := rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)
	// Add celestia wallet to validators (initial setup would have skipped this since the commands didn't exist yet)
	for _, node := range chains.RollchainChain.Validators {
		stdout, stderr, err := node.ExecBin(ctx, "keys", "tiablob", "restore", chains.CelestiaWallets[0].Mnemonic)
		require.NoError(t, err, "stdout: %s, stderr: %s", stdout, stderr)
		t.Log(string(stdout), string(stderr))
	}

	err = testutil.WaitForBlocks(timeoutCtx, 20, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	latestProvenHeight := rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)

	require.Greater(t, latestProvenHeight, origProvenHeight)
}

// TestUpgradeGenesisRestart performs a genesis restart upgrade from the latest height
// Performs a genesis restart, verifies block production, verifies blocks are proved, and adds a fullnode via blocksync
// Genesis restart also tests tiablob's ExportGenesis and then InitGenesis
// go test -timeout 10m -v -run TestUpgradeGenesisRestart . -count 1
func TestUpgradeGenesisRestart(t *testing.T) {
	ctx := context.Background()
	chains := setup.StartCelestiaAndRollchains(t, ctx, 1)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	err = chains.RollchainChain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	state, err := chains.RollchainChain.ExportState(ctx, -1)
	require.NoError(t, err, "error exporting state")

	for _, node := range chains.RollchainChain.Nodes() {
		rollchain.ResetState(t, ctx, node)

		err = node.OverwriteGenesisFile(ctx, []byte(state))
		require.NoError(t, err)
	}

	err = chains.RollchainChain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting node(s)")

	provedHeightAtGenesis := rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	// Ensure blocks are producing
	err = testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	latestProvedHeight := rollchain.GetProvenHeight(t, ctx, chains.RollchainChain)

	// After waiting 10 blocks, proved height must have increased
	require.Greater(t, latestProvedHeight, provedHeightAtGenesis)

	// Add a new full node, with state sync configured, it should sync from block 200
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"override-namespace": "rc_demo0",
			},
		},
	}, 1)
	require.NoError(t, err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	// wait for 10 blocks, new full node should be fully caught up
	err = testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}
}
