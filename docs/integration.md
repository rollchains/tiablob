# Integration

Follow these steps to integrate the tiablob module into your application. If you are using [spawn](https://github.com/rollchains/spawn) to create a new application, simply include the tiablob module to have it pre-wired in your application.

A fully integrated example application is available in this repository under the [simapp](/simapp/) directory which can be used as a reference.

## `app.go` wiring

In your main application file, typically named `app.go`, incorporate the following to wire up the tiablob module

1. Imports

Within the imported packages, add the `tiablob` dependencies.

```golang
import (
    // ...
	nodens "github.com/rollchains/tiablob/celestia-node/share"
	appns "github.com/rollchains/tiablob/celestia/namespace"

	"github.com/rollchains/tiablob"
	tiablobkeeper "github.com/rollchains/tiablob/keeper"
	tiablobmodule "github.com/rollchains/tiablob/module"
	tiablobrelayer "github.com/rollchains/tiablob/relayer"
)
```

2. Configuration Constants

After the imports, declare the constant variables required by `tiablob.

```golang
const (
    // ...

	// namespace identifier for this rollchain on Celestia
	// TODO: Change me
	CelestiaNamespace = "rc_demo"

	// publish blocks to celestia every n blocks.
	publishToCelestiaBlockInterval = 10
)
```

3. Keeper and Relayer declaration

Inside of the `ChainApp` struct, the struct which satisfies the cosmos-sdk `runtime.AppI` interface, add the required `tiablob` runtime fields.

```golang
type ChainApp struct {
	// ...
	// Rollchains Celestia Publish
	TiaBlobKeeper  *tiablobkeeper.Keeper
	TiaBlobRelayer *tiablobrelayer.Relayer
	// ...
}
```

4. Initialize the `tiablob` Keeper and Relayer

Within the `NewChainApp` method, the constructor for the app, initialize the `tiablob` components.

```golang
func NewChainApp(
    // ...
) *ChainApp {
    // ...

    // NOTE: pre-existing code, add parameter.
    keys := storetypes.NewKVStoreKeys(
		// ...

        // Register tiablob Store
		tiablob.StoreKey,
	)

    // ...
	nodeNamespace, err := nodens.NewBlobNamespaceV0([]byte(CelestiaNamespace))
	if err != nil {
		panic(err)
	}
    // TODO: make sure this is after the `app.StakingKeeper` is initialized.
    // Initialize rollchains tiablob keeper
    app.TiaBlobKeeper = tiablobkeeper.NewKeeper(
		appCodec,
		appOpts,
		runtime.NewKVStoreService(keys[tiablob.StoreKey]),
		app.StakingKeeper,
		app.UpgradeKeeper,
		keys[tiablob.StoreKey],
		publishToCelestiaBlockInterval,
		nodeNamespace,
	)

    // Initialize rollchains tiablob relayer
	app.TiaBlobRelayer, err = tiablobrelayer.NewRelayer(
		logger,
		appCodec,
		appOpts,
		appns.MustNewV0([]byte(CelestiaNamespace)),
		filepath.Join(homePath, "keys"),
		publishToCelestiaBlockInterval,
	)
	if err != nil {
		panic(err)
	}

    // Connect relayer to keeper. Must be done after relayer is created.
	app.TiaBlobKeeper.SetRelayer(app.TiaBlobRelayer)

    // Rollchains tiablob proposal handling
	tiaBlobProposalHandler := tiablobkeeper.NewProofOfBlobProposalHandler(
		app.TiaBlobKeeper,
		AppSpecificPrepareProposalHandler(), // i.e. baseapp.NoOpPrepareProposal()
		AppSpecificProcessProposalHandler(), // i.e. baseapp.NoOpProcessProposal()
	)
	bApp.SetPrepareProposal(tiaBlobProposalHandler.PrepareProposal)
	bApp.SetProcessProposal(tiaBlobProposalHandler.ProcessProposal)

    // ...

    // NOTE: pre-existing code, add parameter.
	app.ModuleManager = module.NewManager(
        // ...

        // Register tiablob module
		tiablobmodule.NewAppModule(appCodec, app.TiaBlobKeeper),
	)

    // NOTE: pre-existing code, add parameter.
    app.ModuleManager.SetOrderBeginBlockers(
        // ...

        // tiablob begin blocker can be last
		tiablob.ModuleName,
	)

    // NOTE: pre-existing code, add parameter.
    app.ModuleManager.SetOrderEndBlockers(
        // ...

        // tiablob end blocker can be last
		tiablob.ModuleName,
	)

    // NOTE: pre-existing code, add parameter.
    genesisModuleOrder := []string{
        // ...

        // tiablob genesis module order can be last
        tiablob.ModuleName,
    }

	// NOTE: register tiablob snapshot extension
	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions(
			tiablobkeeper.NewTiablobSnapshotter(app.CommitMultiStore(), app.TiaBlobKeeper),
		)
		// ...
	}

    // ...
}
```

5. Integrate relayer into FinalizeBlock

The `tiablob` relayer needs to be notified when the rollchain blocks are committed so that it is aware of the latest height of the chain. In `FinalizeBlock`, rather than returning `app.BaseApp.FinalizeBlock(req)`, add error handling to that call and notify the relayer afterward.

```golang
func (app *ChainApp) FinalizeBlock(req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
    // ...

	res, err := app.BaseApp.FinalizeBlock(req)
	if err != nil {
		return res, err
	}

	app.TiaBlobRelayer.NotifyCommitHeight(req.Height)

	return res, nil
}
```

6. Integrate `tiablob` PreBlocker

The `tiablob` PreBlocker must be called in the app's PreBlocker.

```golang
func (app *ChainApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	err := app.TiaBlobKeeper.PreBlocker(ctx, req)
	if err != nil {
		return nil, err
	}
	return app.ModuleManager.PreBlock(ctx)
}
```

7. Integrate relayer startup

The relayer needs to query blocks using the client context in order to package them and publish to Celestia. The relayer also needs to be started with some initial values that must be queried from the app after the app has started. Add the following in `RegisterNodeService`.

```golang
func (app *ChainApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)

	app.TiaBlobRelayer.SetClientContext(clientCtx)

	go app.TiaBlobRelayer.Start()
}
```

## `commands.go` wiring

In your application commands file, typically named `cmd/$APP/commands.go`, incorporate the following to wire up the tiablob module CLI commands.

1. Imports

Within the imported packages, add the `tiablob` and `tiasync` dependencies.

```golang
import (
    // ...
	tiablobcli "github.com/rollchains/tiablob/client/cli"
	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/tiasync"
)
```

2. Init App Config

The `app.toml` configuration file needs to be extended with the `tiasync` and `tiablob` relayer options.

```golang
func initAppConfig() (string, interface{}) {
    // Embed the serverconfig.Config in a new type and extend it with the celestia config.
	type CustomAppConfig struct {
		serverconfig.Config

		Celestia *relayer.CelestiaConfig `mapstructure:"celestia"`
		Tiasync  *tiasync.TiasyncConfig  `mapstructure:"tiasync"`
	}

    // ...

	customAppConfig := CustomAppConfig{
		Config:   *srvCfg,
		Celestia: &relayer.DefaultCelestiaConfig,
		Tiasync:  &tiasync.DefaultTiasyncConfig,
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate + relayer.DefaultConfigTemplate + tiasync.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}
```

3. Init Root Command

The keys command needs to be extended with the `tiablob` keys CLI commands to enable Celestia key management.

```golang
func initRootCmd(
    // ...
) {
	// ...

    // Add these two lines to extend the keys command with the tiablob keys commands.
	keysCmd := keys.Commands()
	keysCmd.AddCommand(tiablobcli.NewKeysCmd())

	// Existing code, only modifying one parameter.
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		keysCmd, // replace keys.Commands() here with this
	)
}
```

4. Add start command post setup function to kick off a `tiasync` go routine.

```golang
func AddCommands(
	// ...
) {
	// ...
	startCmd := server.StartCmdWithOptions(appCreator, defaultNodeHome, server.StartCmdOptions{
		PostSetup: func(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group) error {
			go tiasync.TiasyncRoutine(svrCtx, clientCtx, app.CelestiaNamespace)
			return nil

		},
	})
	addStartFlags(startCmd)
	rootCmd.AddCommand(
		startCmd,
		// ...
	)
}
```

## `root.go` wiring

In your application root file, typically named `cmd/$APP/root.go`, incorporate the following to wire up tiasync.

1. Imports

Within the imported packages, add the `tiasync` dependency.

```golang
import (
    // ...
	"github.com/rollchains/tiablob/tiasync"
)
```

2. Tiasync PersistentPreRunE command

```golang
func NewRootCmd() *cobra.Command {
	// ...
	rootCmd := &cobra.Command{
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// ...
			if err := server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig); err != nil {
				return err
			}

			srvCtx := server.GetServerContextFromCmd(cmd)
			tiasync.TiasyncPrerunRoutine(version.AppName, cmd, srvCtx)

			return nil
		},
	}
	// ...
}
```

## Existing chains

Existing chains will have an additional step. An upgrade handler will be needed to set the proven height to the upgrade height.

## Complete

The `tiablob` module is now ready to be used in your application.
