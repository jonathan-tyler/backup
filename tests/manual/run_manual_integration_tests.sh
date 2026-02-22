#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${ROOT_DIR}"

if [[ "$(uname -s)" == "Linux" ]]; then
  export BACKUP_BINARY="${BACKUP_BINARY:-${ROOT_DIR}/out/backup-linux-amd64}"
  TEST_BINARY="${TEST_BINARY:-${ROOT_DIR}/out/itest-manifest-linux-amd64}"
  export WSL_DISTRO_NAME="${WSL_DISTRO_NAME:-devcontainer}"
else
  export BACKUP_BINARY="${BACKUP_BINARY:-${ROOT_DIR}/out/backup-windows-amd64.exe}"
  TEST_BINARY="${TEST_BINARY:-${ROOT_DIR}/out/itest-manifest-windows-amd64.exe}"
fi

if command -v go >/dev/null 2>&1; then
  echo "Go detected. Building binaries before running manual integration tests..." >&2
  "${ROOT_DIR}/tests/manual/build_binaries.sh"
fi

if [[ ! -x "${BACKUP_BINARY}" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "Warning: Go is not installed and required binaries are missing." >&2
    echo "Build in the dev container first: tests/manual/build_binaries.sh" >&2
  fi
  echo "BACKUP_BINARY is not executable: ${BACKUP_BINARY}" >&2
  exit 1
fi

if [[ ! -x "${TEST_BINARY}" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "Warning: Go is not installed and required binaries are missing." >&2
    echo "Build in the dev container first: tests/manual/build_binaries.sh" >&2
  fi
  echo "TEST_BINARY is not executable: ${TEST_BINARY}" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Warning: using prebuilt binaries; they may be out of date. Build in the dev container first if needed (tests/manual/build_binaries.sh)." >&2
fi

export BACKUP_ITEST_PAUSE="${BACKUP_ITEST_PAUSE:-1}"

if [[ -z "${RESTIC_PASSWORD:-}" ]]; then
  export RESTIC_PASSWORD="integration-test-password"
fi

echo
echo "Running consolidated integration manifest test with pause setting BACKUP_ITEST_PAUSE=${BACKUP_ITEST_PAUSE}"
echo "Using backup binary: ${BACKUP_BINARY}"
echo "Using integration test binary: ${TEST_BINARY}"
echo

"${TEST_BINARY}" -test.v -test.run TestIntegrationManifestAllCases
