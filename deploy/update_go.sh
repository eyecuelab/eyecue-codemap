#!/bin/bash

# Note: this script is meant to be sourced vs. executed because it exports PATH

version=1.17.6

echo "Installing Go $version ..."

curl -sSLO https://golang.org/dl/go$version.linux-amd64.tar.gz

sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go$version.linux-amd64.tar.gz

export PATH="/usr/local/go/bin:$PATH"
go version
