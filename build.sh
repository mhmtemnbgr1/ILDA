#!/usr/bin/env bash
# Cross-compiles system-critters for common platforms into ./dist
# Usage:  ./build.sh
set -euo pipefail

name="system-critters"
mkdir -p dist

build() { # $1=GOOS $2=GOARCH $3=output
  echo "building $3..."
  GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w" -o "dist/$3" .
}

build windows amd64 "$name-windows-amd64.exe"
build linux   amd64 "$name-linux-amd64"
build linux   arm64 "$name-linux-arm64"
build darwin  amd64 "$name-macos-amd64"
build darwin  arm64 "$name-macos-arm64"

echo "done -> ./dist"
