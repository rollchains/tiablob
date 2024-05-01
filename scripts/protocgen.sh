#!/usr/bin/env bash

set -eux

cd proto
buf mod update
cd ..

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    # this regex checks if a proto file has its go_package set to github.com/rollchains/tiablob/...
    # gogo proto files SHOULD ONLY be generated if this is false
    # we don't want gogo proto to run for proto files which are natively built for google.golang.org/protobuf
    if grep -q "option go_package" "$file" && grep -H -o -c 'option go_package.*github.com/rollchains/tiablob/api' "$file" | grep -q ':0$'; then
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

echo "Generating pulsar proto code"
buf generate --template buf.gen.pulsar.yaml

cd ..

cp -r github.com/rollchains/tiablob/* ./
rm -rf api && mkdir api
mv rollchains/tiablob/* ./api
rm -rf github.com rollchains

# Hack to avoid "tendermine/crypto/keys.proto" namespace conflict registered by both:
# "buf.build/gen/go/tendermint/tendermint/protocolbuffers/go/tendermint/crypto"
# "cosmossdk.io/api/tendermint/crypto"
rm -rf api/lightclients api/v1/abci.pulsar.go api/v1/genesis.pulsar.go