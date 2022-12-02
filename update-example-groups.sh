#!/bin/bash

{
    echo example-groups.js
    echo example-groups.md
} | go run . --stdin
