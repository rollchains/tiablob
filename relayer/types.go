package relayer

import (
	"github.com/rollchains/tiablob/lightclients/celestia"
)

type InjectedData struct {
	CreateClient *celestia.CreateClient `json:"create_client,omitempty"`
	Headers      []*celestia.Header     `json:"headers,omitempty"`
	Proofs       []*celestia.BlobProof      `json:"proofs,omitempty"`
}

func (d InjectedData) IsEmpty() bool {
	if d.CreateClient != nil || len(d.Headers) != 0 || len(d.Proofs) != 0 {
		return false
	}
	return true
}