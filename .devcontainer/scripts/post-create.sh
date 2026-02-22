#!/usr/bin/env bash
set -euo pipefail

install -d -m 0755 /home/developer/.cache /home/developer/go /home/developer/go/bin

export GOBIN=/home/developer/go/bin
# Allow Go to auto-select a newer toolchain when a tool requires it.
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

# Keep x/tools utilities on known-good versions for stable devcontainer setup.
X_TOOLS_VERSION="v0.31.0"
GOPLS_VERSION="v0.21.1"

install_optional_go_tool() {
	local tool="$1"
	local version="$2"

	echo "Installing ${tool}@${version}..."
	# Limit parallelism to reduce compiler fault risk in constrained containers.
	if ! env GOMAXPROCS=1 go install -v "${tool}@${version}"; then
		echo "Warning: failed to install ${tool}@${version}."
		return 1
	fi

	return 0
}

echo "Installing Go tools..."
# gopls is pinned separately because its release cadence differs from x/tools tags.
install_optional_go_tool golang.org/x/tools/gopls "${GOPLS_VERSION}" \
	|| install_optional_go_tool golang.org/x/tools/gopls latest \
	|| echo "Warning: gopls is unavailable in this container session."

# Do not fail container creation if optional editor tooling cannot be installed.
install_optional_go_tool golang.org/x/tools/cmd/goimports "${X_TOOLS_VERSION}" \
	|| install_optional_go_tool golang.org/x/tools/cmd/goimports latest \
	|| echo "Warning: goimports is unavailable in this container session."

install_optional_go_tool github.com/go-delve/delve/cmd/dlv latest \
	|| echo "Warning: dlv is unavailable in this container session."

install_optional_go_tool honnef.co/go/tools/cmd/staticcheck latest \
	|| echo "Warning: staticcheck is unavailable in this container session."

echo "Downloading Go module dependencies..."
go mod download

echo "Go devcontainer setup complete."
