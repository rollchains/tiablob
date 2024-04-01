package relayer

import (
	"context"
)

// checkForNewBlockProofs will query Celestia for new block proofs and cache them to be included in the next proposal that this validator proposes.
func (r *Relayer) checkForNewBlockProofs(ctx context.Context) {
	// celestiaHeight, err := r.provider.QueryLatestHeight(ctx)
	// if err != nil {
	// 	r.logger.Error("Error querying latest height from Celestia", "error", err)
	// 	return
	// }

	//for i := r.latestProvenHeight + 1; i <= r.latestCommitHeight; i++ {

	// blob := blob.New(namespace.PayForBlobNamespace, []byte(fmt.Sprintf("TODO populate with block data for height i: %d", i)), 0)

	// // generate share commitment for height i
	// commitment, err := inclusion.CreateCommitment(blob, merkle.HashFromByteSlices, 64)
	// if err != nil {
	// 	r.logger.Error("error creating commitment for height", "height", i, "error", err)
	// 	break
	// }

	// // TODO confirm all of these details. Looks like the share commitment may not be able to be queried in the typical way
	// res, err := r.provider.QueryABCI(ctx, abci.RequestQuery{
	// 	Path:   "store/blob/key",
	// 	Data:   commitment,
	// 	Height: celestiaHeight,
	// 	Prove:  true,
	// })
	// if err != nil {
	// 	r.logger.Error("error querying share commitment proof for height", "height", i, "error", err)
	// 	break
	// }

	// if res.Code != 0 {
	// 	r.logger.Error("error querying share commitment proof for height", "height", i, "code", res.Code, "log", res.Log)
	// 	break
	// }

	// cache the proof
	//r.blockProofCache[i] = res.ProofOps.Ops
	//}
}
