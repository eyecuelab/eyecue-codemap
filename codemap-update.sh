#!/bin/bash

set -euo pipefail

# GCP service account used for Docker pull:
# eyecue-codemap@eyecue-ops.iam.gserviceaccount.com
gcp_auth_json=$(base64 -d <<EOF
ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAiZXllY3VlLW9w
cyIsCiAgInByaXZhdGVfa2V5X2lkIjogImU0Y2ExZWEzNzg0MDg1ODc3MjIxYWRiNjg5MzY0Njk3
ZDM2OGY2ZTIiLAogICJwcml2YXRlX2tleSI6ICItLS0tLUJFR0lOIFBSSVZBVEUgS0VZLS0tLS1c
bk1JSUV2QUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktZd2dnU2lBZ0VBQW9JQkFRQ3hnTFNx
a0x4Yk0rWE5cbkg5TTlOVXpXblAzQmk1aVRxcXFiMkd2TzNuaUFIMlJqZ2J0VnZVWnJ6Y1ZFSlZw
ZXJXTUhrWXprbXJxZExmSGRcbkkzLzlUQWFFZXU4UUhiQVdDTkZNaTEzU0EwOTd2OUNUeWFPSU45
N0NRQXNHemtIWFdUUndZSVIzdk5hLzJkVkJcblRJWUd0aEY5Y2pkYkpKVmV3RGpRNkxPQkJLdVJJ
QmFhREUvMWoxb3h0MXhBdWZsNUxXcjFNUStSMWJMUU4rLzBcbkxiWDNxdXYzY2tvWngyZ3lwMkx3
eG4xY09uQVpuRkI3bVFycEp4ckp0WWFjbEdzbU4wN0RIRHhaQW0zcElDdklcbmVvOHpSOVc0eUV0
aEVLZ3BrN3k1Uk1FMU80a0N3RVFGc2ZQcjkycFVGdDBFN2xtbVRMaERDZ1diRktzdkJIbWlcbnJW
bWlmdW45QWdNQkFBRUNnZ0VBQkJySlVyNXBGS1RWRmhLNzFFa0V2MXUvQkhoQmJYeS9XRDQvYy9l
eUNxaHZcbkxsdmtKSjN0WUhYUUwzeTdvNy9YcitlZVBmVVVCcWwyMDluTHptUXhMR1FaWGlLSm5X
RFQwalRRMVJmdGdSdVVcbjY2MzRnYUJsSHRIVFQzTjZrMHZGU2luNU5qbWdMNnlPWVdXdkhiMmRG
dTBLdWFsVmFPMENBMWEyTy9BTE5aNWhcbisyQkREU2IwSmRnMWtxYnhqWWNVSUZYaU5zVTdJak5j
OVlYSnV5WjEybWRhNTB1RFZMSFZEY1NOMmM4b2szc2FcbjE2OUR4c29UWHg0QVRjMUk3TkxPSnJs
Mi9nL0p1c0ErM2paZGR4UHkwUXNQdkNqZk1KeFdJZnhManUxVEhiZDZcbnFuQTdJL0I5SW8yRmtD
eE1pWWVMcGp0OSsxYmFRTlVjbk84Rk1KaG44d0tCZ1FEWG1zWGs2Zmk1Zy9SODNpai9cbkV0UEZV
UjNXektvSVdGNGFwUUxoN2J5RUgzcnlmYk9sWVM0Wncxa0w3VkFSd1JuY2tIaXhpOHpnV1lWNkdq
TXNcblhRcTdEYm9xano3N0k5T2p3bkVwSm1HT1UydG4ybjhWeXpGdEZaNWlxTTJxTHU5Tk9nSENR
cGdXeUZHVGFJT2pcbnZYckZFQ2dSTXFhMFFCNFhMVVBhRjRqUHN3S0JnUURTd213WmppbmdUOGZl
cGZhTXNMemN1VVhqeHQ1OE1nQ29cbmV1QktqQ01ieElGTWhSNDJmb0dGWVlWK2pEakViMTRxSXE3
czB0eXhVMzRha0NIRzZ3RC81Y0puQVI5SzBIT3pcbmNZUUJZZ3psSzNHSTNMTk1CQmRhZXZPNlVk
VlR6RjRlRlhDVTlWMTBwY0lPOW1RQXI2QlhhQkdqaGU4c2N6QWdcbnZacDI2SWNIandLQmdFZXVp
STdrRHpLMm9XbUdmMERXNUp1Y3JYd0Z6WjQ2cXdiV3g4K1B0L2FCZE9IOFV1YndcbkdXQ3Rad1Ns
SU5MV1RaL2NWSlJLODVHL2tiWVgwZDIxRFdWRldoamVTVVU4RXhoR0JGTjNGRVk2aStJYWJkZzBc
bkZ6bTZUMDlqNmdUajErSG9JRCtTM25mc245cVBpL3k3ZVg3ZE1VVU9md2c4clFSdG96cDJTcUVy
QW9HQWFoWHJcbkM4SC9XaVZPV2NmNEhrRW9ENEpDcDdDR2RNVkdoNGV5TmxQcnFDSjFZdXJ1bGtk
L01vQXdEYzdQRkRGcW1KTDBcbnNjaEJ4aEJjdlVvbmRsVDhIOUtxMCtaQXRndk84VmdHTmh3QW1h
b1FiKytIWUkvK29WQ2FOZ0xTK21jNFNMUktcbkF2Q3VwZlI1aGNhSDk4QnZXUS9OTVI1TmtYWTVs
NEZZcXRuSWZna0NnWUJUZ0gxcjhReHlCd05yS1B4OHB4VlhcbnZuR2hFaE1LQkZvaU5BNERxQytC
NEJ3d2ttTTNBdU55bEZWdGpVZ0RrdnZFY2wvc3VGSkUwZmxrVzloVzBybWtcbjV0OFdMNXIwczdX
SUVWL091L1A3Zzk0TjJGK1c5MjdkV2xEOFIwblg4cGYrNXUyWXdCbWxxOEtDUnhTWjFEbTlcbkg3
UXphRktuSklRMlhqaVhDRnRNd0E9PVxuLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLVxuIiwKICAi
Y2xpZW50X2VtYWlsIjogImV5ZWN1ZS1jb2RlbWFwQGV5ZWN1ZS1vcHMuaWFtLmdzZXJ2aWNlYWNj
b3VudC5jb20iLAogICJjbGllbnRfaWQiOiAiMTA1Mjc2ODgyNjA0NTY2MjM3ODcxIiwKICAiYXV0
aF91cmkiOiAiaHR0cHM6Ly9hY2NvdW50cy5nb29nbGUuY29tL28vb2F1dGgyL2F1dGgiLAogICJ0
b2tlbl91cmkiOiAiaHR0cHM6Ly9vYXV0aDIuZ29vZ2xlYXBpcy5jb20vdG9rZW4iLAogICJhdXRo
X3Byb3ZpZGVyX3g1MDlfY2VydF91cmwiOiAiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vb2F1
dGgyL3YxL2NlcnRzIiwKICAiY2xpZW50X3g1MDlfY2VydF91cmwiOiAiaHR0cHM6Ly93d3cuZ29v
Z2xlYXBpcy5jb20vcm9ib3QvdjEvbWV0YWRhdGEveDUwOS9leWVjdWUtY29kZW1hcCU0MGV5ZWN1
ZS1vcHMuaWFtLmdzZXJ2aWNlYWNjb3VudC5jb20iCn0K
EOF
)

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


    # We'll check for updates once per day
    datefile="$tempdir/$executable.date"
    today=$(date '+%Y-%m-%d')
    if [[ ! -f $datefile || $(< "$datefile") != "$today" ]]; then
        echo 'eyecue-codemap checking for updates ...'
        docker_config_path="$tempdir/eyecue-temp-docker"
        echo "$gcp_auth_json" \
          | docker --config "$docker_config_path" login -u _json_key --password-stdin "https://${image%%/*}" 2>&1 \
          | {
              grep -Ev '^(WARNING! Your password|Configure a credential helper|https://docs.docker.com/engine/reference/commandline/login/#credentials-store|Login Succeeded|$)' \
              || true
            }
        docker --config "$docker_config_path" pull "$image"
        echo -n "$today" > "$datefile"
        rm -rf "$docker_config_path"
    fi

    # Create an unstarted docker container for the image. This allows us to
    # copy the executable file out of the image and into the local machine's temp directory.
    cid=$(docker create "$image" --)

    # Delete the unstarted container at the end of this script (even if it exits with an error).
    cleanup() {
        docker rm -f "$cid" >/dev/null
    }
    trap cleanup EXIT

    # Copy the executable file out of the image.
    docker cp "$cid:/bin/$executable" - | tar -x --directory "$tempdir"

    executable="$tempdir/$executable"
}

kernel=$(uname -s|tr '[:upper:]' '[:lower:]')
image="us-central1-docker.pkg.dev/eyecue-ops/eyecue-codemap/eyecue-codemap-$kernel:latest"
executable=eyecue-codemap # must match the files in the container image
prepare_executable

# ↓↓↓ CUSTOMIZE FROM HERE DOWN ↓↓↓

# run from the root of the repo
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
cd "$script_dir/.."

"$executable" --git "$@"
