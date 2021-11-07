#!/bin/bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
cd "$SCRIPT_DIR" || exit 1

docker build -t us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap .
docker push us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap:latest
