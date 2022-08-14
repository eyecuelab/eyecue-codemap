#!/bin/bash

set -euo pipefail

version="${1:-}"
[[ $version ]] || { echo "must specify version argument"; exit 1; }

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"

# Update Go
. "$script_dir/update_go.sh"

# Linting
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$HOME/.local/bin" v1.48.0
./lint.sh

image_prefix="us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap"

export DOCKER_BUILDKIT=1

docker build . --progress=plain --build-arg VERSION="$version" --target linux-final -t "$image_prefix-linux"
docker build . --progress=plain --build-arg VERSION="$version" --target darwin-final -t "$image_prefix-darwin"

gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin https://us-central1-docker.pkg.dev

docker push "$image_prefix-linux"
docker push "$image_prefix-darwin"
