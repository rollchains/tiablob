package tiasync

/*
tiasync is a cometbft p2p proxy.
It allows a full node to only execute blocks posted to Celestia.
When enabled, proper configurations are verified and cometbft's only p2p connection is to tiasync.
Tiasync's peers are the local cometbft instance and if necessary, at least one upstream peer.
An upstream peer is necessary if either state sync is used or to propagate transactions.
The local cometbft instance must not connect to any other peers than tiasync.
Tiasync is used to manage state sync, block sync, and transaction propagation (mempool).
Block sync is started at either genesis or after state sync. Once block sync has started, the node
will stay in this mode, never moving on to consensus. Tiasync will deliver blocks from Celestia to
the local node using block sync.
*/