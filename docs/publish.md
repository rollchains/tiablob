# Celestia Publish

Tiablob enables publishing block data to Celestia via the proposer. There are a few different methods for this: validators can fund their own accounts on Celestia, or they can be feegranted by an account, either a standalone Celestia account or via an Interchain Account (ICA) which is controlled by the tiablob module.

### All Validators

Regardless of the blob payment method, all validators need to have a key loaded for an account on Celestia that will be used for signing transactions to broadcast to the Celestia chain.

Configure Celestia Address in keyring for posting blocks

```bash
rcd keys tiablob add
# OR
rcd keys tiablob restore "some mnemonic"
```

# Celestia blob payment methods

Choose a payment method based on the needs of your application

## 1: Validator funded
Validators fund their own Celestia addresses for payment

1. Fund Validator Celestia Addresses
https://docs.celestia.org/nodes/mocha-testnet#mocha-testnet-faucet

## 2: Off-chain feegrants
A centrally controlled account fee-grants validator payment accounts

### Centrally controlled account instructions

1. Configure Celestia Address in keyring for feegranting validators.
```bash
rcd keys tiablob add --feegrant
# OR
rcd keys tiablob restore --feegrant "some mnemonic"
```

2. Fund Celestia Feegranter Address
https://docs.celestia.org/nodes/mocha-testnet#mocha-testnet-faucet

3. Enable feegrants for all validators
```bash
rcd tx tiablob feegrant [validator1CelestiaAddress] [validator2CelestiaAddress] ...
```

### Validator instructions

1. Add feegranter address to keyring
```bash
rcd keys tiablob add --feegrant --address celestia1dr3gwf5kulm4e4k0pctwzn0htw6wrvevdgjdlf
```

## 3. Feegrants controlled by ICA

TODO

# Publish logic
Proposers check the block interval, i.e. `publishToCelestiaBlockInterval`, defined in `app.go` for publishing blocks to celestia. If a proposer is on this boundary, they will determine the blocks to publish, inject the pending block heights to be published, and call a non-blocking goroutine publishing those blocks as blobs once their proposal is accepted.

If rollchains halts, we shouldn't publish any blocks for a few blocks to ensure we don't find proofs for existing proofs.

# Retry logic
Proposers will check if pending published blocks have timed-out on non-publishing boundries. If they have, they will re-inject the block heights and publish those blocks again, up to the publishing limit. Pending published blocks will only be retried if they have timed out and there is no proof pending for the respective block. If chains publish every block, expired pending blocks will also be checked.

[Dropped transactions](https://docs.celestia.org/developers/submit-data#submitting-multiple-transactions-in-one-block-from-the-same-account)
```
By default, nodes will drop a transaction if it does not get included in 5 blocks (roughly 1 minute). At this point, the user must resubmit their transaction if they want it to eventually be included.
```
[Transaction resubmission](https://docs.celestia.org/developers/transaction-resubmission)
```
In cases where transactions are not included within a 75-second window, resubmission is necessary.
```

# Backoff logic
Celestia may halt for any number of reasons. If a proposer detects a halt, they will not attempt to publish blocks until the chain has resumed producing blocks. Immediately after celestia resumes producing blocks, congestion may further delay publication. Catchup logic will conservatively attempt to publish one rollchain block per celestia block. Every 10 successfully published block will double the published blocks until the target limit has been reached.
