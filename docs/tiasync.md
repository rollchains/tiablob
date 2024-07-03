# Overview

Tiasync allows a fullnode to sync and execute blocks from Celestia instead of the validator network. It is a Cometbft p2p proxy that sits between the fullnode's Cometbft p2p network and validator network. Individual blocks come from Celestia, state sync snapshots/chunks come from the validator network, and transactions are propagated to the validator network.

## Configuration

The minimum configuration for tiasync is below, but other default values can be overridden. This is in addition to the required Celestia/tiablob configurations. Tiasync will use Cometbft's p2p configurations to peer with the validator network while the fullnode will only peer with tiasync.

app.toml
```toml
[tiasync]
    # turns on tiasync
    enable = true

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