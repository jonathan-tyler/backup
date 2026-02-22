#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_PATH="$(mktemp /tmp/backup-integration-manifest.XXXXXX.test)"
cleanup() {
  rm -f "${BIN_PATH}"
}
trap cleanup EXIT

cd "${ROOT_DIR}"

if [[ "${SKIP_OUT_BUILD:-0}" != "1" ]]; then
  "${ROOT_DIR}/tests/manual/build_binaries.sh"
fi

if [[ "$(uname -s)" == "Linux" ]]; then
  export BACKUP_BINARY="${BACKUP_BINARY:-${ROOT_DIR}/out/backup-linux-amd64}"
  export WSL_DISTRO_NAME="${WSL_DISTRO_NAME:-devcontainer}"
else
  export BACKUP_BINARY="${BACKUP_BINARY:-${ROOT_DIR}/out/backup-windows-amd64.exe}"
fi

if [[ ! -x "${BACKUP_BINARY}" ]]; then
  echo "BACKUP_BINARY is not executable: ${BACKUP_BINARY}" >&2
  exit 1
fi

export BACKUP_ITEST_PAUSE="${BACKUP_ITEST_PAUSE:-1}"

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  export RESTIC_PASSWORD="integration-test-password"
fi

echo "Compiling integration test binary..."
go test -c -tags=integration -o "${BIN_PATH}" ./tests/integration

echo
echo "Running consolidated integration manifest test with pause setting BACKUP_ITEST_PAUSE=${BACKUP_ITEST_PAUSE}"
echo "Using backup binary: ${BACKUP_BINARY}"
echo

"${BIN_PATH}" -test.v -test.run TestIntegrationManifestAllCases
