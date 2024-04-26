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

