#!/bin/bash
go_version=$(go version 2>/dev/null | awk -F'[ .]' '{print $4}')
if [[ -z "$go_version" ]] || [[ "$go_version" -lt 21 ]]; then
    echo "Golang is either not installed or its version is less than 1.21.0"
    echo "https://go.dev/doc/install"
    exit 1
else
    mkdir -p binary
    rm -f /binary/groundseg
    arch=$(uname -m); [[ "$arch" == "aarch64" ]] && arc="arm64" || arc="amd64"
    GOOS=linux GOARCH=$arc CGO_ENABLED=0 go build ../goseg -o binary/groundseg
fi
