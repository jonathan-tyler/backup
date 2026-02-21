#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${ROOT_DIR}/out"

mkdir -p "${OUT_DIR}"
cd "${ROOT_DIR}"

echo "Building Linux binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${OUT_DIR}/backup-linux-amd64" .

echo "Building Windows binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${OUT_DIR}/backup-windows-amd64.exe" .

echo
echo "Build artifacts:"
ls -lh "${OUT_DIR}"
