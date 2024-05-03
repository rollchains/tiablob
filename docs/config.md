# Configuration

The `tiablob` configuration is located in `config/app.toml` and is used for connecting to the celestia network.

```toml
[celestia]
    # RPC URL of celestia-app node for posting block data, querying proofs & light blocks
	app-rpc-url = "https://rpc.celestia-app.strange.love"

	# RPC Timeout for transaction broadcasts and queries to celestia-app node
	app-rpc-timeout = "30s"

	# Celestia chain id
	chain-id = "celestia-1"

	# Gas price to pay for celestia transactions
	gas-prices = "0.01utia"

	# Gas adjustment for celestia transactions
	gas-adjustment = 1.0

	# RPC URL of celestia-node for querying proofs
	node-rpc-url = "https://rpc.celestia-node.strange.love:26658"

	# Auth token for celestia-node RPC, n/a if --rpc.skip-auth is used on start
	node-auth-token = "auth-token"
	
	# Query celestia for new block proofs this often
	proof-query-interval = "12s"

	# Only flush at most this many blob proofs in an injected tx per block proposal
	# Must be greater than 0 and less than 100, proofs are roughly 1KB each
	# tiablob will try to aggregate multiple blobs published at the same height w/ a single proof
	max-flush-size = 32
```