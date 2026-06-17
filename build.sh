#!/bin/bash
#
# GetRoute Build Script
# Cross-compiles GetRoute for all target platforms.
#
# Usage:
#   ./build.sh          # Build all platforms
#   ./build.sh darwin   # Build macOS only
#   ./build.sh linux    # Build Linux only
#   ./build.sh windows  # Build Windows only
#

set -e

BIN_DIR="bin"
CMD_DIR="./cmd/getroute"
BIN_NAME="GetRoute"
LDFLAGS="-ldflags=-s -w"

# Ensure Go 1.24+ is available.
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')
if [ "$(printf '%s\n' "1.24" "$GO_VERSION" | sort -V | head -1)" != "1.24" ]; then
    echo "Error: Go 1.24+ required, found $GO_VERSION"
    exit 1
fi

echo "=== GetRoute Build Script ==="
echo "Go version: $(go version)"
echo ""

mkdir -p "$BIN_DIR"

build_target() {
    local os=$1
    local arch=$2
    local suffix=$3
    local output="${BIN_DIR}/${BIN_NAME}-${os}-${arch}${suffix}"

    echo "Building ${os}/${arch}..."
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build ${LDFLAGS} -o "$output" "$CMD_DIR"
    echo "  -> $(ls -lh "$output" | awk '{print $5}')  $output"
}

# Platform builds.
build_linux() {
    build_target linux amd64 ""
    build_target linux arm64 ""
}

build_darwin() {
    build_target darwin amd64 ""
    build_target darwin arm64 ""
}

build_windows() {
    build_target windows amd64 ".exe"
}

build_all() {
    build_linux
    build_darwin
    build_windows
}

case "${1:-all}" in
    linux)   build_linux ;;
    darwin)  build_darwin ;;
    macos)   build_darwin ;;
    windows) build_windows ;;
    all)     build_all ;;
    *)
        echo "Usage: $0 [linux|darwin|windows|all]"
        exit 1
        ;;
esac

echo ""
echo "Build complete. Binaries in ${BIN_DIR}/"
ls -lh "$BIN_DIR"
