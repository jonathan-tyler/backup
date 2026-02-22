#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${ROOT_DIR}/out"

mkdir -p "${OUT_DIR}"
cd "${ROOT_DIR}"

echo "Building Linux binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${OUT_DIR}/backup-linux-amd64" .

echo "Building Linux integration test binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -tags=integration -o "${OUT_DIR}/manual-itest-linux-amd64" ./tests/integration

echo "Building Windows binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${OUT_DIR}/backup-windows-amd64.exe" .

echo "Building Windows integration test binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -c -tags=integration -o "${OUT_DIR}/manual-itest-windows-amd64.exe" ./tests/integration

echo
echo "Build artifacts:"
ls -lh "${OUT_DIR}"
