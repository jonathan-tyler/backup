#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_VERSION_FILE="${ROOT_DIR}/scripts/restic-version.yaml"

CONFIG_BASE_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/backup"
CONFIG_FILE="${CONFIG_BASE_DIR}/config.yaml"
RULES_DIR="${CONFIG_BASE_DIR}/rules"

extract_yaml_restic_version() {
  awk -F': *' '/^restic_version:/ {gsub(/["'"'"']/, "", $2); print $2; exit}' "$1"
}

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

is_version_in_list() {
  local version="$1"
  shift
  for item in "$@"; do
    if [[ "$item" == "$version" ]]; then
      return 0
    fi
  done
  return 1
}

get_installed_linux_version() {
  if ! command -v restic >/dev/null 2>&1; then
    return 0
  fi
  restic version | awk 'NR==1 {print $2}'
}

get_installed_windows_version() {
  if ! command -v restic.exe >/dev/null 2>&1; then
    return 0
  fi
  restic.exe version | awk 'NR==1 {print $2}' | tr -d '\r'
}

mkdir -p "${CONFIG_BASE_DIR}" "${RULES_DIR}"

if [[ ! -f "${REPO_VERSION_FILE}" ]]; then
  echo "Error: repository version pin file not found: ${REPO_VERSION_FILE}" >&2
  exit 1
fi

PINNED_REPO_VERSION="$(extract_yaml_restic_version "${REPO_VERSION_FILE}")"
if [[ -z "${PINNED_REPO_VERSION}" ]]; then
  echo "Error: restic_version is empty or missing in ${REPO_VERSION_FILE}" >&2
  exit 1
fi

if [[ ! -f "${CONFIG_FILE}" ]]; then
  cp "${ROOT_DIR}/config.example.yaml" "${CONFIG_FILE}"
  echo "Created config scaffold: ${CONFIG_FILE}" >&2
fi

RESTIC_VERSION="${PINNED_REPO_VERSION}"

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

if ! is_version_in_list "${RESTIC_VERSION}" "${DNF_VERSIONS[@]}"; then
  echo "Error: restic version ${RESTIC_VERSION} is not available via dnf." >&2
  echo "Available dnf versions: ${DNF_VERSIONS[*]}" >&2
  exit 1
fi

SCOOP_VERSION="$(extract_scoop_manifest_version)"
if [[ -z "${SCOOP_VERSION}" ]]; then
  echo "Error: unable to query restic manifest version from scoop.exe." >&2
  exit 1
fi

if [[ "${SCOOP_VERSION}" != "${RESTIC_VERSION}" ]]; then
  echo "Error: version mismatch between pinned config and scoop manifest." >&2
  echo "Pinned: ${RESTIC_VERSION}" >&2
  echo "Scoop manifest: ${SCOOP_VERSION}" >&2
  exit 1
fi

echo "Installing/updating restic ${RESTIC_VERSION} via dnf..." >&2
if ! sudo dnf install -y "restic-${RESTIC_VERSION}"; then
  echo "Error: failed to install restic-${RESTIC_VERSION} via dnf." >&2
  echo "Hint: check available versions with: dnf --showduplicates list restic" >&2
  exit 1
fi

if ! command -v restic >/dev/null 2>&1; then
  echo "Error: restic was not found after dnf install." >&2
  exit 1
fi

INSTALLED_VERSION="$(get_installed_linux_version)"

if [[ -z "${INSTALLED_VERSION}" ]]; then
  echo "Error: unable to determine installed restic version." >&2
  exit 1
fi

if [[ "${INSTALLED_VERSION}" != "${RESTIC_VERSION}" ]]; then
  echo "Error: installed restic version mismatch. Required: ${RESTIC_VERSION}, Installed: ${INSTALLED_VERSION}" >&2
  exit 1
fi

WINDOWS_INSTALLED_VERSION="$(get_installed_windows_version)"
if [[ "${WINDOWS_INSTALLED_VERSION}" != "${RESTIC_VERSION}" ]]; then
  if scoop.exe list restic >/dev/null 2>&1; then
    echo "Switching Windows restic to ${RESTIC_VERSION} via scoop.exe reset..." >&2
    if ! scoop.exe reset restic "${RESTIC_VERSION}" >/dev/null; then
      echo "Error: failed to reset Windows restic to ${RESTIC_VERSION} with scoop.exe." >&2
      exit 1
    fi
  else
    echo "Installing Windows restic ${RESTIC_VERSION} via scoop.exe..." >&2
    if ! scoop.exe install "restic@${RESTIC_VERSION}" >/dev/null; then
      echo "Error: failed to install Windows restic ${RESTIC_VERSION} with scoop.exe." >&2
      exit 1
    fi
  fi
fi

WINDOWS_INSTALLED_VERSION="$(get_installed_windows_version)"
if [[ "${WINDOWS_INSTALLED_VERSION}" != "${RESTIC_VERSION}" ]]; then
  echo "Error: Windows restic version mismatch. Required: ${RESTIC_VERSION}, Installed: ${WINDOWS_INSTALLED_VERSION:-<not found>}" >&2
  exit 1
fi

create_rule_file_if_missing() {
  local file_path="$1"
  local default_line="$2"
  if [[ ! -f "${file_path}" ]]; then
    printf '# One path/pattern per line.\n%s\n' "${default_line}" > "${file_path}"
    echo "Created rules scaffold: ${file_path}" >&2
  fi
}

create_rule_file_if_missing "${RULES_DIR}/wsl.include.daily.txt" "/home/<user>/documents"
create_rule_file_if_missing "${RULES_DIR}/wsl.include.weekly.txt" "/home/<user>/documents"
create_rule_file_if_missing "${RULES_DIR}/wsl.include.monthly.txt" "/home/<user>/documents"
create_rule_file_if_missing "${RULES_DIR}/wsl.exclude.daily.txt" "/home/<user>/.cache"
create_rule_file_if_missing "${RULES_DIR}/wsl.exclude.weekly.txt" "/home/<user>/.cache"
create_rule_file_if_missing "${RULES_DIR}/wsl.exclude.monthly.txt" "/home/<user>/.cache"

create_rule_file_if_missing "${RULES_DIR}/windows.include.daily.txt" "C:\\Users\\<user>\\Documents"
create_rule_file_if_missing "${RULES_DIR}/windows.include.weekly.txt" "C:\\Users\\<user>\\Documents"
create_rule_file_if_missing "${RULES_DIR}/windows.include.monthly.txt" "C:\\Users\\<user>\\Documents"
create_rule_file_if_missing "${RULES_DIR}/windows.exclude.daily.txt" "C:\\Users\\<user>\\AppData\\Local\\Temp"
create_rule_file_if_missing "${RULES_DIR}/windows.exclude.weekly.txt" "C:\\Users\\<user>\\AppData\\Local\\Temp"
create_rule_file_if_missing "${RULES_DIR}/windows.exclude.monthly.txt" "C:\\Users\\<user>\\AppData\\Local\\Temp"

echo
echo "WSL/Fedora and Windows restic are installed and pinned by config file."
echo "Linux version: ${INSTALLED_VERSION}"
echo "Windows version: ${WINDOWS_INSTALLED_VERSION}"
echo "Repo pin: ${REPO_VERSION_FILE}"
echo "Config: ${CONFIG_FILE}"
