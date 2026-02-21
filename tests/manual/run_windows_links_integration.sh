#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_PATH="$(mktemp /tmp/backup-integration-win-links.XXXXXX.test)"
cleanup() {
  rm -f "${BIN_PATH}"
}
trap cleanup EXIT

cd "${ROOT_DIR}"

if [[ "${SKIP_OUT_BUILD:-0}" != "1" ]]; then
  "${ROOT_DIR}/tests/manual/build_binaries.sh"
fi

export BACKUP_BINARY="${BACKUP_BINARY:-${ROOT_DIR}/out/backup-windows-amd64.exe}"

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  export RESTIC_PASSWORD="integration-test-password"
fi

echo "Compiling integration test binary..."
go test -c -tags=integration -o "${BIN_PATH}" ./tests/integration

echo
echo "Running Windows links integration test..."
echo "Note: this test skips unless running on Windows with required link privileges."
echo "Using backup binary: ${BACKUP_BINARY}"
echo

"${BIN_PATH}" -test.v -test.run TestIntegrationWindowsLinksFlow
