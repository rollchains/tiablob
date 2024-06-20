# Overview

Tiasync allows a fullnode to sync and execute blocks from Celestia instead of the validator network. It is a cometbft p2p proxy that sits between the fullnode's cometbft p2p network and validator network. Individual blocks come from Celestia, state sync snapshots/chunks come from the validator network, and transactions are propagated to the validator network.

## Configuration

The minimum configurations for tiasync are below. These are in addition to the required Celestia/tiablob configurations. When tiasync is enabled, these minimum configurations will be verified so that the fullnode cannot connect to the validator network directly.

config.toml
```toml
[p2p]
    # Listen address must be localhost only, port can be changed
    laddr = "tcp://127.0.0.1:26777"

    # Must have no seeds
    seeds = ""

    # Must have no persistent peers
    persistent_peers = ""

    # Address book must not be strict
    addr_book_strict = false

    # Duplicate ips should be allowed (local fullnode and tiasync)
    allow_duplicate_ip = true

    # Pex must be false/disabled
    pex = false
```

app.toml
```toml
[tiasync]
    # turns on tiasync
    enable = true

    # Address for tiasync to listen for incoming connection from the validator network
    laddr = "tcp://0.0.0.0:26656"

    # Peers that are upstream from this fullnode and on the validator network
    # optional when only syncing from genesis, but necessary for state sync and transaction propagation
    upstream-peers = "upstream-node1-id@ip:port,upstream-node2-id@ip:port"

```

## Block sync

Tiasync's primary mode is block sync. It uses block sync to deliver blocks to its fullnode. These blocks are retrieved from Celestia and verified before being delivered to the local fullnode's cometbft instance. On the initial startup, the fullnode may start in state sync mode, transitioning to block sync mode once it has its snapshot. The fullnode will never enter consensus while tiasync is enabled.

## State sync

Initial state can come from the validator network instead of from genesis using state sync. Currently, tiasync only supports a single upstream peer for state sync. If there are multiple upstream peers, the first connected upstream peer is used. At least one upstream peer is required to use state sync.

## Transaction propagation

Transactions sent to the fullnode will be propagated to the validator network through tiasync. At least one upstream peer is required to propagate transactions. Pex is used in tiasync to increase distribution.

## Existing chains upgrading with tiablob

### Genesis restart

If the chain performs a genesis restart upgrade, tiasync can be used to sync from that genesis state.

### In-place store migration

Upgrades using the recommended in-place store migration to add tiablob support requires new fullnodes syncing from Celestia to start from a height no less than the upgrade height. This can be accomplished by using state sync or having that fullnode sync from the validator network and then switch to syncing from Celestia.

## Vote extensions

Currently, both tiablob and tiasync do not support chains that have vote extensions enabled. Support is expected soon.