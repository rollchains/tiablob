package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/rollchains/tiablob"
	"github.com/rollchains/tiablob/relayer"
	"github.com/rollchains/tiablob/relayer/cosmos"
)

const (
	FlagFeeGrant = "feegrant"
	FlagPubKey   = "pubkey"
	FlagAddress  = "address"
	FlagLedger   = "ledger"
	FlagCoinType = "coin-type"
)

// NewKeysCmd returns a root CLI command handler for all x/tiablob keys commands.
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
		Short: "Generate a key for posting blocks or feegranting on Celestia",
		Args:  cobra.NoArgs,
		Example: fmt.Sprintf(` 
$ %s keys tiablob add
`, version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := relayer.CelestiaConfigFromAppOpts(serverCtx.Viper)

			keyDir := filepath.Join(clientCtx.HomeDir, "keys")
			provider, err := cosmos.NewProvider(cfg.RpcURL, keyDir, 0)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			keyName := relayer.CelestiaPublishKeyName
			feeGrant, _ := cmd.Flags().GetBool(FlagFeeGrant)
			if feeGrant {
				keyName = relayer.CelestiaFeegrantKeyName
			}

			exists := provider.KeyExists(keyName)
			if exists {
				return fmt.Errorf("key already exists")
			}

			address, _ := cmd.Flags().GetString(FlagAddress)
			pubKey, _ := cmd.Flags().GetString(FlagPubKey)
			ledger, _ := cmd.Flags().GetBool(FlagLedger)
			coinType, _ := cmd.Flags().GetUint32(FlagCoinType)

			var ko *cosmos.KeyOutput

			if address != "" {
				accountInfo, err := provider.AccountInfo(cmd.Context(), address)
				if err != nil {
					return fmt.Errorf("failed to query account info: %w", err)
				}

				ko, err = provider.SaveOfflineKey(keyName, accountInfo.GetPubKey())
				if err != nil {
					return err
				}
			} else if pubKey != "" {
				var pk cryptotypes.PubKey
				if err = clientCtx.Codec.UnmarshalInterfaceJSON([]byte(pubKey), &pk); err != nil {
					return err
				}
				ko, err = provider.SaveOfflineKey(keyName, pk)
				if err != nil {
					return err
				}
			} else if ledger {
				ko, err = provider.SaveLedgerKey(keyName, coinType, "celestia")
				if err != nil {
					return err
				}
			} else {
				ko, err = provider.AddKey(keyName, coinType, "secp256k1")
				if err != nil {
					return err
				}

				fmt.Printf(`WRITE THIS MNEMONIC PHRASE DOWN AND KEEP IT SAFE. IT IS THE ONLY WAY TO RECOVER THE ACCOUNT
  Mnemonic: "%s"
  CoinType: %d
`, ko.Mnemonic, coinType)
			}

			b32, err := bech32.ConvertAndEncode("celestia", ko.Account)
			if err != nil {
				return err
			}

			fmt.Printf("  Address: %s\n", b32)

			return nil
		},
	}

	cmd.Flags().BoolP(FlagFeeGrant, "f", false, "Feegranter key")
	cmd.Flags().Bool(FlagLedger, false, "Store a key for a Ledger device")
	cmd.Flags().String(FlagPubKey, "", "Specify a public key to add")
	cmd.Flags().String(FlagAddress, "", "Specify an address to query from Celestia to save as an offline key")
	cmd.Flags().Uint32(FlagCoinType, 118, "Coin type for key derivation")

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
			provider, err := cosmos.NewProvider("", keyDir, 0)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			keyName := relayer.CelestiaPublishKeyName
			feeGrant, _ := cmd.Flags().GetBool(FlagFeeGrant)
			if feeGrant {
				keyName = relayer.CelestiaFeegrantKeyName
			}

			exists := provider.KeyExists(keyName)
			if exists {
				return fmt.Errorf("key already exists")
			}

			coinType, _ := cmd.Flags().GetUint32(FlagCoinType)

			res, err := provider.RestoreKey(keyName, args[0], coinType, "secp256k1")
			if err != nil {
				return err
			}

			b32, err := bech32.ConvertAndEncode("celestia", res)
			if err != nil {
				return err
			}

			fmt.Printf(`CoinType: %d
Address: %s
`, coinType, b32)

			return nil
		},
	}

	cmd.Flags().BoolP(FlagFeeGrant, "f", false, "Feegranter key")
	cmd.Flags().Uint32(FlagCoinType, 118, "Coin type for key derivation")

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
			provider, err := cosmos.NewProvider("", keyDir, 0)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			keyName := relayer.CelestiaPublishKeyName
			feeGrant, _ := cmd.Flags().GetBool(FlagFeeGrant)
			if feeGrant {
				keyName = relayer.CelestiaFeegrantKeyName
			}

			res, err := provider.ShowAddress(keyName, "celestia")
			if err != nil {
				return err
			}

			fmt.Println(res)

			return nil
		},
	}

	cmd.Flags().BoolP(FlagFeeGrant, "f", false, "Feegranter key")

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
			provider, err := cosmos.NewProvider("", keyDir, 0)
			if err != nil {
				return err
			}

			if err := provider.CreateKeystore(); err != nil {
				return err
			}

			keyName := relayer.CelestiaPublishKeyName
			feeGrant, _ := cmd.Flags().GetBool(FlagFeeGrant)
			if feeGrant {
				keyName = relayer.CelestiaFeegrantKeyName
			}

			if err := provider.DeleteKey(keyName); err != nil {
				return err
			}

			fmt.Println("Key deleted")

			return nil
		},
	}

	cmd.Flags().BoolP(FlagFeeGrant, "f", false, "Feegranter key")

	return cmd
}
