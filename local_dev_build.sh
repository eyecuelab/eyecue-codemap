#!/bin/bash

# This is an ad-hoc script used for local development. Not part of the automated build process.

set -euo pipefail

image_prefix="us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap"

export DOCKER_BUILDKIT=1

docker build . --build-arg VERSION="dev" --target linux-final -t "$image_prefix-linux"
