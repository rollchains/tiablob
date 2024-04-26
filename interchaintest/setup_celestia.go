package e2e

import (
	"context"
	"fmt"
	"testing"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	blobtypes "github.com/rollchains/tiablob/celestia/blob/types"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

var (
	numCelestiaVals      = 1
	numCelestiaFullNodes = 0
	celestiaCoinDecimals = int64(6)
	celestiaChainID      = "celestia-1"
	celestiaNodeHome     = "/var/cosmos-chain/celestia-node"
	celestiaNodeType     = "bridge"

	CelestiaChainConfig = ibc.ChainConfig{
		Name:           "celestia",
		Denom:          "utia",
		Type:           "cosmos",
		GasPrices:      "0.01utia",
		TrustingPeriod: "500h",
		EncodingConfig: func() *moduletestutil.TestEncodingConfig {
			cfg := cosmos.DefaultEncoding()
			blobtypes.RegisterInterfaces(cfg.InterfaceRegistry)
			return &cfg
		}(),
		SkipGenTx:           false,
		CoinDecimals:        &celestiaCoinDecimals,
		AdditionalStartArgs: []string{"--grpc.enable"},
		ChainID:             celestiaChainID,
		Bin:                 "celestia-appd",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/strangelove-ventures/heighliner/celestia",
				Version:    "v1.8.0",
				UidGid:     "1025:1025",
			},
		},
		Bech32Prefix:  "celestia",
		CoinType:      "118",
		GasAdjustment: 1.5,
	}

	CelestiaChainSpec = interchaintest.ChainSpec{
		Name:          CelestiaChainConfig.Name,
		ChainName:     CelestiaChainConfig.Name,
		Version:       CelestiaChainConfig.Images[0].Version,
		ChainConfig:   CelestiaChainConfig,
		NumValidators: &numCelestiaVals,
		NumFullNodes:  &numCelestiaFullNodes,
	}
)

func StartCelestiaNode(t *testing.T, ctx context.Context, celestiaChain *cosmos.CosmosChain, client *client.Client, network string) *cosmos.SidecarProcess {
	celestiaVal0 := celestiaChain.GetNode()

	genesisHash := getGenesisBlockHash(t, ctx, celestiaVal0)
	fmt.Println("genesisHash: ", genesisHash)

	// Create chain sidecar process (this could also be per validator, but we only need one and it will sync with val-0)
	err := celestiaChain.NewSidecarProcess(
		ctx,
		false,           // preStart
		"celestia-node", //processName
		t.Name(),        // testName
		client,          //docker client
		network,         // docker network
		ibc.DockerImage{
			Repository: "celestia-node",
			Version:    "local",
			//Repository: "ghcr.io/strangelove-ventures/heighliner/celestia-node",
			//Version: "v0.13.1",
			UidGid: "1025:1025",
		},
		celestiaNodeHome, // home dir
		0,                // index
		[]string{"26650", "26658", "26659", "2121"}, // ports
		[]string{"celestia", celestiaNodeType, "start",
			"--node.store", celestiaNodeHome,
			"--gateway",
			//"--keyring.accname", "celnode",
			"--core.ip", celestiaVal0.HostName(),
			"--gateway.addr", "0.0.0.0",
			"--rpc.addr", "0.0.0.0",
			"--rpc.skip-auth",
		}, // start cmd
		[]string{
			fmt.Sprintf("CELESTIA_CUSTOM=%s:%s", celestiaChainID, genesisHash),
			fmt.Sprintf("NODE_STORE=%s", celestiaNodeHome),
			fmt.Sprintf("NODE_TYPE=%s", celestiaNodeType),
			fmt.Sprintf("P2P_NETWORK=%s", celestiaChainID),
		}, //env
	)
	require.NoError(t, err, "failed to create celestia-node sidecar")

	sc := celestiaChain.Sidecars[len(celestiaChain.Sidecars)-1]
	err = sc.CreateContainer(ctx)
	require.NoError(t, err, "failed to create sidecar container")

	_, _, err = sc.Exec(
		ctx,
		[]string{
			"celestia", celestiaNodeType, "init",
			"--p2p.network", celestiaChainID,
			"--node.store", celestiaNodeHome,
			"--rpc.skip-auth",
		}, // cmd
		[]string{
			fmt.Sprintf("CELESTIA_CUSTOM=%s:%s", celestiaChainID, genesisHash),
			fmt.Sprintf("NODE_STORE=%s", celestiaNodeHome),
			fmt.Sprintf("NODE_TYPE=%s", celestiaNodeType),
			fmt.Sprintf("P2P_NETWORK=%s", celestiaChainID),
		}, //env
	)
	require.NoError(t, err, "failed to init celestia-node")

	/*_, _, err = sc.Exec(
		ctx,
		[]string{
			"sh",
			"-c",
			fmt.Sprintf(`echo %q | cel-key add celnode --recover --keyring-backend test --node.type %s --home %s --p2p.network %s`,
			"kick raven pave wild outdoor dismiss happy start lunch discover job evil code trim network emerge summer mad army vacant chest birth subject seek",
			celestiaNodeType,
			celestiaNodeHome,
			celestiaChainID,
			),
		}, // cmd
		[]string{
			fmt.Sprintf("CELESTIA_CUSTOM=%s:%s", celestiaChainID, genesisHash),
			fmt.Sprintf("NODE_STORE=%s", celestiaNodeHome),
			fmt.Sprintf("NODE_TYPE=%s", celestiaNodeType),
			fmt.Sprintf("P2P_NETWORK=%s", celestiaChainID),
		}, //env
	)
	require.NoError(t, err, "failed to init celestia-node")*/

	err = sc.StartContainer(ctx)
	require.NoError(t, err, "failed to start sidecar container")

	return sc
}

func getGenesisBlockHash(t *testing.T, ctx context.Context, node *cosmos.ChainNode) string {
	height := int64(1)
	block, err := node.Client.Block(ctx, &height)
	require.NoError(t, err, "failed getting block 1")

	return block.BlockID.Hash.String()
}
