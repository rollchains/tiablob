package celestia

/*
celestia light client is based on the ibc-go 07-tendermint and 07-celestia light client.
It has no dependencies on ibc-go. It uses json encoding instead of proto encoding due
to the proto dependencies not using pulsar while tiablob does.
Then encoding may change in the future, but the additional overhead should not be
significant since only 1 client will be present.
Pruning is TBD, we may only require the latest consensus state.
The consensus state root also uses the data hash instead of app hash.
*/
