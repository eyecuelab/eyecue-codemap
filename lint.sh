#!/bin/bash

set -euo pipefail

command -v golangci-lint >/dev/null 2>&1 || { echo 'Please install https://golangci-lint.run/'; exit 1; }
golangci-lint --version
golangci-lint run --timeout 10m
