#!/bin/sh
# Builds the pstack-setup binary for each supported platform into bin/.
# Run this after changing anything under setup/, then commit bin/.
set -eu

cd "$(dirname "$0")/.."
mkdir -p bin

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
	os=${target%/*}
	arch=${target#*/}
	out="bin/pstack-setup-$os-$arch"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
		go build -trimpath -ldflags="-s -w" -o "$out" ./setup
	echo "built $out ($(wc -c <"$out" | tr -d ' ') bytes)"
done
