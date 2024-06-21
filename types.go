package tiablob

func (d MsgInjectedData) IsEmpty() bool {
	if d.CreateClient != nil || len(d.Headers) != 0 || len(d.Proofs) != 0 || len(d.PendingBlocks.BlockHeights) != 0 {
		return false
	}
	return true
}
