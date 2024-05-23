package node

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"go.uber.org/zap"
)

// Celestia node client is use for celestia node apis, i.e. "blob get-all"
type CelestiaNodeClient struct {
	log *zap.Logger

	sc        *cosmos.SidecarProcess
	nodeStore string
}

func NewCelestiaNodeClient(log *zap.Logger, sc *cosmos.SidecarProcess, nodeStore string) *CelestiaNodeClient {
	return &CelestiaNodeClient{
		log:       log,
		sc:        sc,
		nodeStore: nodeStore,
	}
}

func (n *CelestiaNodeClient) GetAllBlobs(ctx context.Context, height uint64, namespace string) ([]jsonBlob, error) {
	cmd := []string{
		"celestia", "blob", "get-all", fmt.Sprint(height), namespace,
		"--node.store", n.nodeStore,
		"--url", fmt.Sprintf("http://%s:26658", n.sc.HostName()),
		"--base64",
	}
	stdout, _, err := n.sc.Exec(ctx, cmd, []string{})
	if err != nil {
		return nil, err
	}

	var blobs jsonGetAllBlobs
	err = json.Unmarshal(stdout, &blobs)
	if err != nil {
		return []jsonBlob{}, nil
	}

	return blobs.Blobs, nil
}

type jsonGetAllBlobs struct {
	Blobs []jsonBlob `json:"result"`
}

type jsonBlob struct {
	Namespace    []byte `json:"namespace"`
	Data         []byte `json:"data"`
	ShareVersion uint32 `json:"share_version"`
	Commitment   []byte `json:"commitment"`
	Index        int    `json:"index"`
}
