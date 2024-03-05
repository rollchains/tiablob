package cli

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"cosmossdk.io/core/address"

	"github.com/rollchains/tiablob"
)

// NewTxCmd returns a root CLI command handler for all x/tiablob transaction commands.
func NewTxCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        tiablob.ModuleName,
		Short:                      tiablob.ModuleName + "transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSetCelestiaAddrCmd(ac),
		NewProveBlockCmd(ac),
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

// NewProveBlockCmd returns a CLI command handler for creating a prove block transaction.
func NewProveBlockCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prove-block [height] [base64-proof]",
		Short: "Prove a block",
		Args:  cobra.ExactArgs(1),
		Long:  `Prove a block by providing the height and the base64 encoded share commitment proof from Celestia`,
		Example: fmt.Sprintf(`
$ %s tx tiablob prove-block 25 base64proof --from keyname
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

			height, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse height: %w", err)
			}

			proof, err := base64.StdEncoding.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("failed to decode proof: %w", err)
			}

			msg := &tiablob.MsgProveBlock{
				ValidatorAddress: valAddr,
				Height:           height,
				Proof:            proof,
			}

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}
