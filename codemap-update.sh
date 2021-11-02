#!/bin/bash

set -euo pipefail

prepare_executable() {
    # Find the system temp directory so we can put our executable and date files there.
    tempdir=""
    for dir in "${TMPDIR:-}" "${TEMP:-}" /tmp; do
        if [[ -d $dir ]]; then
            tempdir="$dir"
            break
        fi
    done
    [[ -z $tempdir ]] && { echo "ERROR: could not locate system temp directory"; exit 1; }

    # We'll check for updates once per day (unless running in CircleCI)
    datefile="$tempdir/$executable_prefix.date"
    today=$(date '+%Y-%m-%d')
    if [[ -z ${CIRCLECI:-} && ( ! -f $datefile || $(< "$datefile") != "$today" ) ]]; then
        echo 'Checking for updates...'
        docker pull "$image"
        echo -n "$today" > "$datefile"
    fi

    kernel=$(uname -s|tr '[:upper:]' '[:lower:]')

    # Create an unstarted docker container for the image. This allows us to
    # copy the executable file out of the image and into the local machine's temp directory.
    cid=$(docker create "$image" --)

    # Delete the unstarted container at the end of this script (even if it exits with an error).
    cleanup() {
        docker rm -f "$cid" >/dev/null
    }
    trap cleanup EXIT

    # Copy the executable file out of the image.
    docker cp "$cid:/bin/$executable_prefix-$kernel" - | tar -x --directory "$tempdir"

    executable="$tempdir/$executable_prefix-$kernel"
}

image=us.gcr.io/eyecue-io/eyecue-codemap:latest
executable_prefix=eyecue-codemap # must match the files in the container image
prepare_executable

# ↓↓↓ CUSTOMIZE FROM HERE DOWN ↓↓↓

# TODO: customize find command to your repo
# TODO: cd to the root of your repo if needed
find . -type d \( -path ./.git -o -path ./node_modules \) -prune -o -type f -print | "$executable" "$@"
