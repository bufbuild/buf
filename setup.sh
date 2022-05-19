#!/usr/bin/env bash

set -e

mkdir -p connectbuf
go build -o connectbuf ./...
pushd connectbuf
cp buf cbuf
popd