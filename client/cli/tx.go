package cli

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/version"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	"github.com/rollchains/tiablob"
	blobtypes "github.com/rollchains/tiablob/celestia/blob/types"
	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/relayer/celestia"
)

// NewTxCmd returns a root CLI command handler for all x/tiablob transaction commands.
func NewTxCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        tiablob.ModuleName,
		Short:                      tiablob.ModuleName + " transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSetCelestiaAddrCmd(ac),
		NewFeegrantCmd(),
	)

	return txCmd
}

// NewSetCelestiaAddrCmd returns a CLI command handler for creating a MsgSetCelestiaAddress transaction.
func NewSetCelestiaAddrCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-celestia-address [celestia-addr]",
		Short: "Set the Celestia address for the validator",
		Args:  cobra.ExactArgs(1),
		Long:  `Set the Celestia address of the validator for feegranting on Celestia.`,
		Example: fmt.Sprintf(`
$ %s tx tiablob set-celestia-address celestia1jzv52ewect8ntvwjs2za087yzl6y3smf5etf3n --from keyname
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			valAddr, err := ac.BytesToString(clientCtx.GetFromAddress())
			if err != nil {
				return fmt.Errorf("failed to convert validator address: %w", err)
			}

			msg := &tiablob.MsgSetCelestiaAddress{
				ValidatorAddress: valAddr,
				CelestiaAddress:  args[0],
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

// NewFeegrantCmd returns a CLI command handler for creating a feegrant transaction on Celestia.
func NewFeegrantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feegrant [address1] [address2] ...",
		Short: "Feegrant an address for posting blocks to Celestia",
		Args:  cobra.MinimumNArgs(1),
		Example: fmt.Sprintf(`
$ %s tx tiablob feegrant celestia1jzv52ewect8ntvwjs2za087yzl6y3smf5etf3n
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				hrp, _, err := bech32.DecodeAndConvert(arg)
				if err != nil {
					return fmt.Errorf("failed to decode feegrantee address: %w", err)
				}

				if hrp != "celestia" {
					return fmt.Errorf("feegrantee address must have celestia prefix")
				}
			}

			serverCtx := server.GetServerContextFromCmd(cmd)

			cfg := relayer.CelestiaConfigFromAppOpts(serverCtx.Viper)

			provider, err := celestia.NewProvider(cfg.AppRpcURL, filepath.Join(serverCtx.Config.RootDir, "keys"), cfg.AppRpcTimeout, cfg.ChainID)
			if err != nil {
				return fmt.Errorf("failed to create cosmos provider: %w", err)
			}

			if err := provider.CreateKeystore(); err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			granter, err := provider.ShowAddress(relayer.CelestiaFeegrantKeyName, "celestia")
			if err != nil {
				return fmt.Errorf("failed to get granter address: %w", err)
			}

			chainID, err := provider.QueryChainID(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to query chain ID: %w", err)
			}

			exp := time.Now().Add(365 * 24 * time.Hour)

			allowance := &feegrant.BasicAllowance{
				SpendLimit: sdk.NewCoins(sdk.NewCoin("utia", math.NewInt(1000000))), // TODO make configurable
				Expiration: &exp,
			}

			allowanceAny, err := codectypes.NewAnyWithValue(allowance)
			if err != nil {
				return fmt.Errorf("failed to create allowance any: %w", err)
			}

			allowedMsgAllowance, err := codectypes.NewAnyWithValue(&feegrant.AllowedMsgAllowance{
				Allowance: allowanceAny,
				AllowedMessages: []string{
					blobtypes.URLMsgPayForBlobs,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to create allowed messages allowance any: %w", err)
			}

			msgs := make([]sdk.Msg, len(args))
			for i, grantee := range args {
				msgs[i] = &feegrant.MsgGrantAllowance{
					Granter:   granter,
					Grantee:   grantee,
					Allowance: allowedMsgAllowance,
				}
			}

			res, err := provider.SignAndBroadcast(
				cmd.Context(),
				chainID,
				cfg.GasPrice,
				1.3,
				0,
				"celestia",
				relayer.CelestiaFeegrantKeyName,
				nil,
				msgs,
				"",
			)
			if err != nil {
				fmt.Println(err)
				return fmt.Errorf("failed to feegrant address: %w", err)
			}

			fmt.Printf("Feegranted address(es) with tx hash %s\n", hex.EncodeToString(res.Hash))

			return nil
		},
	}

	return cmd
}
