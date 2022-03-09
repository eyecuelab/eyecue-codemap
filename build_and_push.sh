#!/bin/bash

set -euo pipefail

version="${1:-}"
[[ $version ]] || { echo "must specify version argument"; exit 1; }

# Linting
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$HOME/.local/bin" v1.43.0
./lint.sh

image_prefix="us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap"

docker build . --build-arg VERSION="$version" --target linux-final -t "$image_prefix-linux"
docker build . --build-arg VERSION="$version" --target darwin-final -t "$image_prefix-darwin"

echo "$GOOGLE_AUTH_JSON" | docker login -u _json_key --password-stdin https://us-central1-docker.pkg.dev

docker push "$image_prefix-linux"
docker push "$image_prefix-darwin"
