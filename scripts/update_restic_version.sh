#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_VERSION_FILE="${ROOT_DIR}/scripts/restic-version.yaml"
CONFIG_BASE_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/backup"
CONFIG_FILE="${CONFIG_BASE_DIR}/config.yaml"
INSTALL_SCRIPT="${ROOT_DIR}/scripts/install_restic_wsl_fedora.sh"

extract_dnf_versions() {
  dnf --showduplicates --quiet list restic 2>/dev/null \
    | awk '/^restic[[:space:]]/ {print $2}' \
    | sed 's/-.*$//' \
    | sort -V \
    | uniq
}

extract_scoop_manifest_version() {
  scoop.exe cat restic 2>/dev/null \
    | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]\+\)".*/\1/p' \
    | head -n1 \
    | tr -d '\r'
}

set_yaml_restic_version() {
  local file_path="$1"
  local version="$2"
  if grep -q '^restic_version:' "${file_path}"; then
    sed -i -E "s/^restic_version: .*/restic_version: \"${version}\"/" "${file_path}"
  else
    printf 'restic_version: "%s"\n\n' "${version}" | cat - "${file_path}" > "${file_path}.tmp"
    mv "${file_path}.tmp" "${file_path}"
  fi
}

if [[ ! -f "${REPO_VERSION_FILE}" ]]; then
  echo "Error: repository version pin file not found: ${REPO_VERSION_FILE}" >&2
  exit 1
fi

mkdir -p "${CONFIG_BASE_DIR}"
if [[ ! -f "${CONFIG_FILE}" ]]; then
  cp "${ROOT_DIR}/config.example.yaml" "${CONFIG_FILE}"
  echo "Created config scaffold: ${CONFIG_FILE}" >&2
fi

if ! command -v dnf >/dev/null 2>&1; then
  echo "Error: dnf not found. This script is intended for Fedora-based WSL." >&2
  exit 1
fi

if ! command -v scoop.exe >/dev/null 2>&1; then
  echo "Error: scoop.exe not found from WSL PATH. Ensure Scoop is installed on Windows and accessible." >&2
  exit 1
fi

mapfile -t DNF_VERSIONS < <(extract_dnf_versions)
if [[ ${#DNF_VERSIONS[@]} -eq 0 ]]; then
  echo "Error: unable to query restic versions from dnf." >&2
  exit 1
fi

LATEST_DNF_VERSION="${DNF_VERSIONS[-1]}"
SCOOP_VERSION="$(extract_scoop_manifest_version)"

if [[ -z "${SCOOP_VERSION}" ]]; then
  echo "Error: unable to query restic manifest version from scoop.exe." >&2
  exit 1
fi

if [[ "${LATEST_DNF_VERSION}" != "${SCOOP_VERSION}" ]]; then
  echo "Error: latest dnf restic version and scoop manifest version differ." >&2
  echo "dnf latest: ${LATEST_DNF_VERSION}" >&2
  echo "scoop manifest: ${SCOOP_VERSION}" >&2
  exit 1
fi

set_yaml_restic_version "${REPO_VERSION_FILE}" "${LATEST_DNF_VERSION}"
echo "Updated ${REPO_VERSION_FILE} to restic_version: ${LATEST_DNF_VERSION}" >&2

"${INSTALL_SCRIPT}"
