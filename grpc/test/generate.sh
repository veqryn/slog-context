#!/usr/bin/env bash

# This is a spec/documentation first approach: We write the Protobuf doc first,
# then generate both client sdk's and app stubs/interfaces from it.
set -euxo pipefail
cd "$(dirname "$0")"

# Remove old generated files
rm -rf ./gen

# Use buf to create the protobuf binaries in desired languages
# https://buf.build/docs/tutorials/getting-started-with-buf-cli#update-directory-path-and-build-module
buf dep update
buf generate --template buf.gen.yaml
