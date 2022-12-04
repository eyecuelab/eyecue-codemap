#!/bin/bash

go run . --stdin "$@" <<-EOF
example-groups.js
example-groups.md
EOF
