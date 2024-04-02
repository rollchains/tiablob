package cosmos

import (
	"crypto/rand"
	"errors"
	"os"

	ckeys "github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/go-bip39"
)

const ethereumCoinType = uint32(60)

var (
	// SupportedAlgorithms defines the list of signing algorithms used on Evmos:
	//  - secp256k1     (Cosmos)
	//  - sr25519		(Cosmos)
	//  - eth_secp256k1 (Ethereum, Injective)
	SupportedAlgorithms = keyring.SigningAlgoList{hd.Secp256k1 /*sr25519.Sr25519, ethermint.EthSecp256k1, injective.EthSecp256k1*/}
	// SupportedAlgorithmsLedger defines the list of signing algorithms used on Evmos for the Ledger device:
	//  - secp256k1     (Cosmos)
	//  - sr25519		(Cosmos)
	//  - eth_secp256k1 (Ethereum, Injective)
	SupportedAlgorithmsLedger = keyring.SigningAlgoList{hd.Secp256k1 /*sr25519.Sr25519, ethermint.EthSecp256k1, injective.EthSecp256k1*/}
)

// KeyringAlgoOptions defines a function keys options for the ethereum Secp256k1 curve.
// It supports secp256k1 and eth_secp256k1 keys for accounts.
func KeyringAlgoOptions() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
	}
}

// CreateKeystore initializes a new instance of a keyring at the specified path in the local filesystem.
func (cc *CosmosProvider) CreateKeystore() error {
	keybase, err := keyring.New("tiablob", "test", cc.keyDir, rand.Reader, cc.cdc.Marshaler, KeyringAlgoOptions())
	if err != nil {
		return err
	}
	cc.keybase = keybase
	return nil
}

func (cc *CosmosProvider) LoadKeystore(keyring keyring.Keyring) {
	cc.keybase = keyring
}

// KeystoreCreated returns true if there is an existing keystore instance at the specified path, it returns false otherwise.
func (cc *CosmosProvider) KeystoreCreated() bool {
	if _, err := os.Stat(cc.keyDir); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return cc.keybase != nil
}

// AddKey generates a new mnemonic which is then converted to a private key and BIP-39 HD Path and persists it to the keystore.
// It fails if there is an existing key with the same address.
func (cc *CosmosProvider) AddKey(name string, coinType uint32, signingAlgorithm string) (output *KeyOutput, err error) {
	ko, err := cc.KeyAddOrRestore(name, coinType, signingAlgorithm)
	if err != nil {
		return nil, err
	}
	return ko, nil
}

// RestoreKey converts a mnemonic to a private key and BIP-39 HD Path and persists it to the keystore.
// It fails if there is an existing key with the same address.
func (cc *CosmosProvider) RestoreKey(name, mnemonic string, coinType uint32, signingAlgorithm string) (address sdk.AccAddress, err error) {
	ko, err := cc.KeyAddOrRestore(name, coinType, signingAlgorithm, mnemonic)
	if err != nil {
		return nil, err
	}
	return ko.Account, nil
}

type KeyOutput struct {
	Mnemonic string `json:"mnemonic"`
	CoinType uint32 `json:"coin_type"`
	Account  sdk.AccAddress
}

// KeyAddOrRestore either generates a new mnemonic or uses the specified mnemonic and converts it to a private key
// and BIP-39 HD Path which is then persisted to the keystore. It fails if there is an existing key with the same address.
func (cc *CosmosProvider) KeyAddOrRestore(keyName string, coinType uint32, signingAlgorithm string, mnemonic ...string) (*KeyOutput, error) {
	var mnemonicStr string
	var err error

	var algo keyring.SignatureAlgo
	// switch signingAlgorithm {
	// case string(hd.Sr25519Type):
	// 	algo = sr25519.Sr25519
	// default:
	algo = hd.Secp256k1
	//}

	if len(mnemonic) > 0 {
		mnemonicStr = mnemonic[0]
	} else {
		mnemonicStr, err = CreateMnemonic()
		if err != nil {
			return nil, err
		}
	}

	//if coinType == ethereumCoinType {
	// algo = keyring.SignatureAlgo(ethermint.EthSecp256k1)
	// for _, codec := range cc.PCfg.ExtraCodecs {
	// 	if codec == "injective" {
	// 		algo = keyring.SignatureAlgo(injective.EthSecp256k1)
	// 	}
	//
	//}

	info, err := cc.keybase.NewAccount(keyName, mnemonicStr, "", hd.CreateHDPath(coinType, 0, 0).String(), algo)
	if err != nil {
		return nil, err
	}

	addr, err := info.GetAddress()
	if err != nil {
		return nil, err
	}

	return &KeyOutput{Mnemonic: mnemonicStr, Account: addr}, nil
}

// SaveOfflineKey persists a public key to the keystore with the specified name.
func (cc *CosmosProvider) SaveOfflineKey(keyName string, pubKey cryptotypes.PubKey) (*KeyOutput, error) {
	info, err := cc.keybase.SaveOfflineKey(keyName, pubKey)
	if err != nil {
		return nil, err
	}

	addr, err := info.GetAddress()
	if err != nil {
		return nil, err
	}

	return &KeyOutput{Account: addr}, nil
}

// SaveLedgerKey persists a ledger key to the keystore with the specified name.
func (cc *CosmosProvider) SaveLedgerKey(keyName string, coinType uint32, bech32Prefix string) (*KeyOutput, error) {
	info, err := cc.keybase.SaveLedgerKey(keyName, hd.Secp256k1, bech32Prefix, coinType, 0, 0)
	if err != nil {
		return nil, err
	}

	addr, err := info.GetAddress()
	if err != nil {
		return nil, err
	}

	return &KeyOutput{Account: addr}, nil
}

// ShowAddress retrieves a key by name from the keystore and returns the bech32 encoded string representation of that key.
func (cc *CosmosProvider) ShowAddress(name string, bech32Prefix string) (address string, err error) {
	info, err := cc.keybase.Key(name)
	if err != nil {
		return "", err
	}

	addr, err := info.GetAddress()
	if err != nil {
		return "", err
	}

	out, err := bech32.ConvertAndEncode(bech32Prefix, addr)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ListAddresses returns a map of bech32 encoded strings representing all keys currently in the keystore.
func (cc *CosmosProvider) ListAddresses(bech32Prefix string) (map[string]string, error) {
	out := map[string]string{}
	info, err := cc.keybase.List()
	if err != nil {
		return nil, err
	}
	for _, k := range info {
		a, err := k.GetAddress()
		if err != nil {
			return nil, err
		}

		addr, err := bech32.ConvertAndEncode(bech32Prefix, a)
		if err != nil {
			return nil, err
		}
		out[k.Name] = addr
	}
	return out, nil
}

// DeleteKey removes a key from the keystore for the specified name.
func (cc *CosmosProvider) DeleteKey(name string) error {
	if err := cc.keybase.Delete(name); err != nil {
		return err
	}
	return nil
}

// KeyExists returns true if a key with the specified name exists in the keystore, it returns false otherwise.
func (cc *CosmosProvider) KeyExists(name string) bool {
	k, err := cc.keybase.Key(name)
	if err != nil {
		return false
	}

	return k.Name == name

}

// ExportPrivKeyArmor returns a private key in ASCII armored format.
// It returns an error if the key does not exist or a wrong encryption passphrase is supplied.
func (cc *CosmosProvider) ExportPrivKeyArmor(keyName string) (armor string, err error) {
	return cc.keybase.ExportPrivKeyArmor(keyName, ckeys.DefaultKeyPass)
}

// GetKeyAddress returns the account address representation for the currently configured key.
func (cc *CosmosProvider) GetKeyAddress(key string) (sdk.AccAddress, error) {
	info, err := cc.keybase.Key(key)
	if err != nil {
		return nil, err
	}

	addr, err := info.GetAddress()
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// CreateMnemonic generates a new mnemonic.
func CreateMnemonic() (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func (cc *CosmosProvider) KeyFromKeyOrAddress(keyOrAddress string) (string, error) {
	switch {
	case cc.KeyExists(keyOrAddress):
		return keyOrAddress, nil
	default:
		_, acc, err := bech32.DecodeAndConvert(keyOrAddress)
		if err != nil {
			return "", err
		}
		kr, err := cc.keybase.KeyByAddress(sdk.AccAddress(acc))
		if err != nil {
			return "", err
		}
		return kr.Name, nil
	}
}
