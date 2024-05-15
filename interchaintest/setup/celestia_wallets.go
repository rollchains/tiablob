package setup

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

type CelestiaWallet struct {
	Address  string
	Mnemonic string
}

func BuildCelestiaWallets(t *testing.T, numWallets int) []CelestiaWallet {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kr := keyring.NewInMemory(cdc)

	celestiaWallets := make([]CelestiaWallet, numWallets)
	for i := 0; i < numWallets; i++ {
		info, mnemonic, err := kr.NewMnemonic(
			fmt.Sprintf("rollchain%d", i),
			keyring.English,
			hd.CreateHDPath(118, 0, 0).String(),
			"", // Empty passphrase.
			hd.Secp256k1,
		)
		require.NoError(t, err)

		addrBytes, err := info.GetAddress()
		require.NoError(t, err)

		celestiaWallets[i].Mnemonic = mnemonic
		celestiaWallets[i].Address = types.MustBech32ifyAddressBytes("celestia", addrBytes)
	}

	return celestiaWallets
}
