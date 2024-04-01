package cosmos

import (
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"

	client "github.com/strangelove-ventures/cometbft-client/client"
)

var _ gogogrpc.ClientConn = &CosmosProvider{}

var protoCodec = encoding.GetCodec(proto.Name)

var accountSeqRegex = regexp.MustCompile("account sequence mismatch, expected ([0-9]+), got ([0-9]+)")

type CosmosProvider struct {
	cdc       Codec
	rpcClient RPCClient
	keybase   keyring.Keyring

	keyDir string

	// the map key is the TX signer, which can either be 'default' (provider key) or a feegrantee
	// the purpose of the map is to lock on the signer from TX creation through submission,
	// thus making TX sequencing errors less likely.
	walletStateMap map[string]*WalletState
	walletStateMu  sync.Mutex
}

type WalletState struct {
	NextAccountSequence uint64
	Mu                  sync.Mutex
}

func (ws *WalletState) updateNextAccountSequence(seq uint64) {
	if seq > ws.NextAccountSequence {
		ws.NextAccountSequence = seq
	}
}

func (cc *CosmosProvider) EnsureWalletState(address string) *WalletState {
	cc.walletStateMu.Lock()
	defer cc.walletStateMu.Unlock()

	if ws, ok := cc.walletStateMap[address]; ok {
		return ws
	}

	ws := &WalletState{}
	cc.walletStateMap[address] = ws
	return ws
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider(rpcURL string, keyDir string) (*CosmosProvider, error) {
	rpcClient, err := client.NewClient(rpcURL, 5*time.Second)
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		cdc:            makeCodec(ModuleBasics),
		rpcClient:      NewRPCClient(rpcClient),
		keyDir:         keyDir,
		walletStateMap: make(map[string]*WalletState),
	}

	return cp, nil
}

// handleAccountSequenceMismatchError will parse the error string, e.g.:
// "account sequence mismatch, expected 10, got 9: incorrect account sequence"
// and update the next account sequence with the expected value.
func (ws *WalletState) HandleAccountSequenceMismatchError(err error) {
	matches := accountSeqRegex.FindStringSubmatch(err.Error())
	if len(matches) == 0 {
		return
	}
	nextSeq, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return
	}
	ws.NextAccountSequence = nextSeq
}
