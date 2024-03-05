package module

import (
	"context"
	"encoding/json"
	"fmt"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"cosmossdk.io/core/appmodule"

	"github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/client/cli"
	"github.com/rollchains/tiablob/keeper"
)

var (
	_ module.AppModule          = AppModule{}
	_ module.AppModuleGenesis   = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 1

type AppModule struct {
	cdc    codec.Codec
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

func NewAppModuleBasic(m AppModule) module.AppModuleBasic {
	return module.CoreAppModuleBasicAdaptor(tiablob.ModuleName, m)
}

// Name returns the rollchain module's name.
func (AppModule) Name() string { return tiablob.ModuleName }

// RegisterLegacyAminoCodec registers the rollchain module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	tiablob.RegisterLegacyAminoCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the rollchain module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := tiablob.RegisterQueryHandlerClient(context.Background(), mux, tiablob.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the rollchain module.
func (am AppModule) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd(am.cdc.InterfaceRegistry().SigningContext().ValidatorAddressCodec())
}

// RegisterInterfaces registers interfaces and implementations of the rollchain module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	tiablob.RegisterInterfaces(registry)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	tiablob.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	tiablob.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	// Register in place module state migration migrations
	// m := keeper.NewMigrator(am.keeper)
	// if err := cfg.RegisterMigration(rollchain.ModuleName, 1, m.Migrate1to2); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", rollchain.ModuleName, err))
	// }
}

// DefaultGenesis returns default genesis state as raw bytes for the module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(tiablob.NewGenesisState())
}

// ValidateGenesis performs genesis state validation for the rollchain module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data tiablob.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", tiablob.ModuleName, err)
	}

	return data.Validate()
}

// InitGenesis performs genesis initialization for the rollchain module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState tiablob.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if err := am.keeper.InitGenesis(ctx, &genesisState); err != nil {
		panic(err)
	}

	return nil
}

// ExportGenesis returns the exported genesis state as raw bytes for the rollchain
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)

	return cdc.MustMarshalJSON(gs)
}

func (am AppModule) BeginBlock(ctx context.Context) error {
	return nil
}
