package cli

import (
	"fmt"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/relayer/cosmos"
	"github.com/spf13/cobra"
)

const keyName = "blob"

// NewTxCmd returns a root CLI command handler for all x/tiablob transaction commands.
func NewKeysCmd() *cobra.Command {
	keysCmd := &cobra.Command{
		Use:                        tiablob.ModuleName,
		Short:                      tiablob.ModuleName + " keys subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	keysCmd.AddCommand(
		NewKeysAddCmd(),
		NewKeysRestoreCmd(),
		NewKeysShowCmd(),
		NewKeysDeleteCmd(),
	)

	return keysCmd
}

// NewKeysAddCmd returns a CLI command handler for creating a key for posting blocks to Celestia.
func NewKeysAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Generate a key for posting blocks to Celestia",
		Args:  cobra.NoArgs,
		Example: fmt.Sprintf(` 
$ %s keys tiablob add
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			keyDir := filepath.Join(clientCtx.HomeDir, "keys")
			provider, err := cosmos.NewProvider("http://localhost:26657", keyDir)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error 1: %s", err)
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error 2: %s", err)
				return err
			}

			exists := provider.KeyExists(keyName)
			if exists {
				return fmt.Errorf("key already exists")
			}

			res, err := provider.AddKey(keyName, 118, "secp256k1")
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error 3: %s", err)
				return err
			}

			b32, err := bech32.ConvertAndEncode("celestia", res.Account)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error 4: %s", err)
				return err
			}

			fmt.Printf(`WRITE THIS MNEMONIC PHRASE DOWN AND KEEP IT SAFE. IT IS THE ONLY WAY TO RECOVER THE ACCOUNT
  Mnemonic: "%s"
  CoinType: %d
  Address: %s
`, res.Mnemonic, 118, b32)

			return nil
		},
	}

	return cmd
}

// NewKeysRestoreCmd returns a CLI command handler for restoring a key for posting blocks to Celestia.
func NewKeysRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [mnemonic]",
		Short: "Generate a key for posting blocks to Celestia",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(` 
$ %s keys tiablob restore "pattern match caution ..."
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			keyDir := filepath.Join(clientCtx.HomeDir, "keys")
			provider, err := cosmos.NewProvider("http://localhost:26657", keyDir)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			exists := provider.KeyExists(keyName)
			if exists {
				return fmt.Errorf("key already exists")
			}

			res, err := provider.RestoreKey(keyName, args[0], 118, "secp256k1")
			if err != nil {
				return err
			}

			b32, err := bech32.ConvertAndEncode("celestia", res)
			if err != nil {
				return err
			}

			fmt.Printf(`CoinType: %d
Address: %s
`, 118, b32)

			return nil
		},
	}

	return cmd
}

// NewKeysShowCmd returns a CLI command handler for showing the key for posting blocks to Celestia.
func NewKeysShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the key for posting blocks to Celestia",
		Args:  cobra.NoArgs,
		Example: fmt.Sprintf(` 
$ %s keys tiablob show
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			keyDir := filepath.Join(clientCtx.HomeDir, "keys")
			provider, err := cosmos.NewProvider("http://localhost:26657", keyDir)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			res, err := provider.ShowAddress(keyName, "celestia")
			if err != nil {
				return err
			}

			fmt.Println(res)

			return nil
		},
	}

	return cmd
}

// NewKeysDeleteCmd returns a CLI command handler for deleting the key for posting blocks to Celestia.
func NewKeysDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete the key for posting blocks to Celestia",
		Args:  cobra.NoArgs,
		Example: fmt.Sprintf(` 
$ %s keys tiablob delete
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			keyDir := filepath.Join(clientCtx.HomeDir, "keys")
			provider, err := cosmos.NewProvider("http://localhost:26657", keyDir)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			if err := provider.DeleteKey(keyName); err != nil {
				return err
			}

			fmt.Println("Key deleted")

			return nil
		},
	}

	return cmd
}
