#!/bin/bash

set -euo pipefail

image_prefix="us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap"

docker build . --target linux-final -t "$image_prefix-linux"
docker build . --target darwin-final -t "$image_prefix-darwin"

echo "$GOOGLE_AUTH_JSON" | docker login -u _json_key --password-stdin https://us-central1-docker.pkg.dev

docker push "$image_prefix-linux"
docker push "$image_prefix-darwin"
