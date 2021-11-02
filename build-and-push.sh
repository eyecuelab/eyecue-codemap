#!/bin/bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
cd "$SCRIPT_DIR" || exit 1

docker build -t us.gcr.io/eyecue-io/eyecue-codemap:latest .
docker push us.gcr.io/eyecue-io/eyecue-codemap:latest
