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
	
	"golang.org/x/sync/errgroup"
)

// go test -timeout 10m -v -run TestTiasyncVeFromGenesis . -count 1
func TestTiasyncVeFromGenesis(t *testing.T) {
	ctx := context.Background()
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 1)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0, "1")
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(2), rollchainChainSpecs)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err := testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks")

	// Add a new full node that syncs from celestia
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(timeoutCtx, 100, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}
}

// TestTiasyncVeStateSync produces >210 blocks (2 snapshots), gets trusted height/hash for fullnode, starts/syncs fullnode with statesync
// verifies all nodes are producing blocks and the state sync'd fullnode does not have blocks earlier than the snapshot height
// go test -timeout 20m -v -run TestTiasyncVeStateSync . -count 1
func TestTiasyncVeStateSync(t *testing.T) {
	ctx := context.Background()
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 1)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0, "1")
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(2), rollchainChainSpecs)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*15)
	defer timeoutCtxCancel()

	// Wait 210 blocks for 2 snapshots, 100 block interval
	err := testutil.WaitForBlocks(timeoutCtx, 210, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks with 2 snapshots")

	// Get a trusted height and hash, wait for 10 blocks for chain to proceed
	trustedHeight, trustedHash := rollchain.GetTrustedHeightAndHash(t, ctx, chains.RollchainChain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 10, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after getting trusted height and hash")

	// Add a new full node, with state sync configured, it should sync from block 200
	// Add a new full node, with state sync configured, it should sync from block 200
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	rollchainVal0Hostname := fmt.Sprintf("%s-val-0-%s", chains.RollchainChain.Config().ChainID, t.Name())
	rollchainVal1Hostname := fmt.Sprintf("%s-val-1-%s", chains.RollchainChain.Config().ChainID, t.Name())
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
			},
		},
		"config/config.toml": testutil.Toml{
			//"log_level": "debug",
			"statesync": testutil.Toml{
				"enable":       true,
				"rpc_servers":  fmt.Sprintf("%s:26657,%s:26657", rollchainVal0Hostname, rollchainVal1Hostname),
				"trust_height": trustedHeight,
				"trust_hash":   trustedHash,
				"trust_period": "1h",
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	// wait for 30 blocks, new full node should be fully caught up
	err = testutil.WaitForBlocks(timeoutCtx, 100, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	// Ensure all nodes have a height greater than 250 (sum of WaitForBlocks, 210+10+30)
	// Full node will state sync from block 200, there should be 5-10 pending blocks, if it gets a few blocks passed 200, we're good
	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}

	// Test assumes no fullnodes sync'd from genesis
	// verify that a block before the snapshot height (200) does not exist
	height := int64(150)
	for _, fn := range chains.RollchainChain.FullNodes {
		_, err = fn.Client.Block(ctx, &height)
		require.Error(t, err)
		require.Contains(t, err.Error(), "height 150 is not available, lowest height is 201")
	}
}

// TestUpgradeTiasyncVeFromGenesis performs an upgrade without a genesis restart
// Pre-reqs: build local2 verison of rollchains with version in makefile set to 2 and upgrade handler setting vote extension enable height
//
//	heighliner build -c rollchain --local -f chains.yaml --go-version 1.22.1 -g local2
//
// go test -timeout 25m -v -run TestUpgradeTiasyncVeFromGenesis . -count 1
func TestUpgradeTiasyncVeFromGenesis(t *testing.T) {
	ctx := context.Background()
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 1)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0, "0")
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(2), rollchainChainSpecs)

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "wallet", userFunds, chains.RollchainChain)
	user := users[0]

	// Add a new full node that syncs from celestia
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	err := chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

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
	var eg errgroup.Group
	for _, val := range chains.RollchainChain.Validators {
		val := val
		eg.Go(func() error {
			if err := val.StopContainer(ctx); err != nil {
				return err
			}
			return val.RemoveContainer(ctx)
		})
	}
	err = eg.Wait()
	require.NoError(t, err, "error stopping val(s)")

	// upgrade version on all nodes
	chains.RollchainChain.UpgradeVersion(ctx, chains.Client, "rollchain", "local2")

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	for _, val := range chains.RollchainChain.Validators {
		val := val
		eg.Go(func() error {
			if err := val.CreateNodeContainer(ctx); err != nil {
				return err
			}
			return val.StartContainer(ctx)
		})
	}
	err = eg.Wait()
	require.NoError(t, err, "error starting val(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*75)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 20, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// Restart the fullnode for the upgrade (node is expected to be halted for the upgrade at this point)
	for _, fn := range chains.RollchainChain.FullNodes {
		fn := fn
		eg.Go(func() error {
			if err := fn.StopContainer(ctx); err != nil {
				return err
			}
			if err := fn.RemoveContainer(ctx); err != nil {
				return err
			}
			if err := fn.CreateNodeContainer(ctx); err != nil {
				return err
			}
			return fn.StartContainer(ctx)
		})
	}
	err = eg.Wait()
	require.Error(t, err)
	fmt.Println("Error fullnode restart1:", err)

// Restart the fullnode for the upgrade (node is expected to be halted for the upgrade at this point)
	var eg2 errgroup.Group
for _, fn := range chains.RollchainChain.FullNodes {
	fn := fn
	eg2.Go(func() error {
		if err := fn.StopContainer(ctx); err != nil {
			return err
		}
		if err := fn.RemoveContainer(ctx); err != nil {
			return err
		}
		if err := fn.CreateNodeContainer(ctx); err != nil {
			return err
		}
		return fn.StartContainer(ctx)
	})
}
err = eg2.Wait()
require.Error(t, err)
fmt.Println("Error fullnode restart2:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(timeoutCtx, 25, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}
}

// TestUpgradeTiasyncVeStateSync produces >210 blocks (2 snapshots), gets trusted height/hash for fullnode, starts/syncs fullnode with statesync
// verifies all nodes are producing blocks and the state sync'd fullnode does not have blocks earlier than the snapshot height
// go test -timeout 20m -v -run TestUpgradeTiasyncVeStateSync . -count 1
func TestUpgradeTiasyncVeStateSync(t *testing.T) {
	ctx := context.Background()
	rollchainChainSpecs := make([]*interchaintest.ChainSpec, 1)
	rollchainChainSpecs[0] = setup.RollchainChainSpec(t.Name(), 2, 0, "rc_demo", 0, "0")
	chains := setup.StartWithSpecs(t, ctx, celestiaChainSpec(2), rollchainChainSpecs)

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

	// UPGRADE FINISHED
	// Wait enough blocks for state sync snapshots

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*15)
	defer timeoutCtxCancel()

	// Wait 210 blocks for 2 snapshots, 100 block interval
	err = testutil.WaitForBlocks(timeoutCtx, 200, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks with 2 snapshots")

	// Get a trusted height and hash, wait for 10 blocks for chain to proceed
	trustedHeight, trustedHash := rollchain.GetTrustedHeightAndHash(t, ctx, chains.RollchainChain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, 2, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after getting trusted height and hash")

	// Add a new full node, with state sync configured, it should sync from block 200
	// Add a new full node, with state sync configured, it should sync from block 200
	celestiaChainID := "celestia-1"
	celestiaAppHostname := fmt.Sprintf("%s-val-0-%s", celestiaChainID, t.Name())            // celestia-1-val-0-TestPublish
	celestiaNodeHostname := fmt.Sprintf("%s-celestia-node-0-%s", celestiaChainID, t.Name()) // celestia-1-celestia-node-0-TestPublish
	rollchainVal0Hostname := fmt.Sprintf("%s-val-0-%s", chains.RollchainChain.Config().ChainID, t.Name())
	rollchainVal1Hostname := fmt.Sprintf("%s-val-1-%s", chains.RollchainChain.Config().ChainID, t.Name())
	err = chains.RollchainChain.AddFullNodes(ctx, testutil.Toml{
		"config/app.toml": testutil.Toml{
			"celestia": testutil.Toml{
				"app-rpc-url":        fmt.Sprintf("http://%s:26657", celestiaAppHostname),
				"node-rpc-url":       fmt.Sprintf("http://%s:26658", celestiaNodeHostname),
				"override-namespace": "rc_demo",
			},
			"tiasync": testutil.Toml{
				"enable":         true,
			},
		},
		"config/config.toml": testutil.Toml{
			//"log_level": "debug",
			"statesync": testutil.Toml{
				"enable":       true,
				"rpc_servers":  fmt.Sprintf("%s:26657,%s:26657", rollchainVal0Hostname, rollchainVal1Hostname),
				"trust_height": trustedHeight,
				"trust_hash":   trustedHash,
				"trust_period": "1h",
			},
		},
	}, 1)
	require.Error(t, err) // expected to fail, always catching up
	fmt.Println("Error:", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*5)
	defer timeoutCtxCancel()

	previousHeight, err := chains.RollchainChain.Height(ctx)
	require.NoError(t, err)

	// wait for 30 blocks, new full node should be fully caught up
	err = testutil.WaitForBlocks(timeoutCtx, 100, chains.RollchainChain)
	require.NoError(t, err, "chain did not produce blocks after adding fullnode")

	// Ensure all nodes have a height greater than 250 (sum of WaitForBlocks, 210+10+30)
	// Full node will state sync from block 200, there should be 5-10 pending blocks, if it gets a few blocks passed 200, we're good
	nodes := chains.RollchainChain.Nodes()
	for _, node := range nodes {
		latestHeight, err := node.Height(ctx)
		require.NoError(t, err)
		t.Log("Node:", node.Name(), "Previous Height:", previousHeight, "Current Height:", latestHeight)
		require.Greater(t, latestHeight, previousHeight, "a node has not increased height enough")
	}

	// Test assumes no fullnodes sync'd from genesis
	// verify that a block before the snapshot height (200) does not exist
	height = int64(150)
	for _, fn := range chains.RollchainChain.FullNodes {
		_, err = fn.Client.Block(ctx, &height)
		require.Error(t, err)
		require.Contains(t, err.Error(), "height 150 is not available, lowest height is 201")
	}


	// Restart the fullnode for the upgrade (node is expected to be halted for the upgrade at this point)
	var eg errgroup.Group
	for _, fn := range chains.RollchainChain.FullNodes {
		fn := fn
		eg.Go(func() error {
			if err := fn.StopContainer(ctx); err != nil {
				return err
			}
			if err := fn.RemoveContainer(ctx); err != nil {
				return err
			}
			if err := fn.CreateNodeContainer(ctx); err != nil {
				return err
			}
			return fn.StartContainer(ctx)
		})
	}
	err = eg.Wait()
	require.Error(t, err)
	fmt.Println("Error fullnode restart1:", err)
}
