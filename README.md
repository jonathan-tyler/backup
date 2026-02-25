# WSL Backup

Thin, predictable wrapper around `restic`, designed for WSL-first workflows and usable as a `wsl-sys-cli` extension.

## What it does

- Runs from WSL and targets cross-platform backup flows (WSL + Windows)
- Uses include/exclude rule files with daily, weekly, and monthly cadences
- Parses report modes (`new`, `excluded`) with placeholder output (not implemented yet)
- Enforces overlap safety checks across profile include paths
- Includes unit, integration, and manual test paths for cross-platform behavior

## Installation

```sh
go build -o backup .
sudo install -m 0755 backup /usr/local/bin/backup
```

Pin/install `restic` for both WSL Fedora and Windows (Scoop):

```sh
scripts/install_restic_wsl_fedora.sh
```

The install script validates the pinned version from `scripts/restic-version.yaml` in both package managers, installs both binaries, and scaffolds config/rules files when missing.

## Configuration

- Default config path: `${XDG_CONFIG_HOME:-~/.config}/backup/config.yaml`
- Optional config override: `BACKUP_CONFIG=/path/to/config.yaml`
- Starter config: [config.example.yaml](config.example.yaml)
- Rule file directory: `~/.config/backup/rules/` (next to config)
- Rule naming: `<profile>.<include|exclude>.<daily|weekly|monthly>.txt`
- Rule format: one path per line (`#` comments allowed)
- Optional per-config overrides: `include_files`, `exclude_files`

## Usage

- This CLI is WSL-only; run it from a WSL shell (not from native Windows or a Dev Container).
- `backup run <cadence>` runs `wsl` and `windows` profiles in parallel.
- Include overlap checks are strict by default and fail the run when overlap is detected.
- Current execution status:
  - `backup run daily` executes restic for both profiles only when config exists and is valid.
  - `backup run weekly|monthly` currently returns scaffold-only output.
  - `backup report ...` modes currently return not-implemented messages.
  - `backup restore <target>` executes `restic restore latest --target <target>` for the WSL profile and requires a config file.

```sh
backup run daily
backup report weekly
backup report weekly new
backup report weekly excluded
backup restore /path/to/target
backup test
```

If installed through `wsl-sys-cli`, run the same commands as `sys backup ...`.

## Development

```sh
go test ./...
go build ./...
```

Integration tests:

```sh
go test ./tests/unit/...
go test -tags=integration ./tests/integration/...
```

Manual integration tests:

- Run `backup test` from WSL.
- Linux manual tests run first, then Windows manual tests, with a review pause between phases.

## Notes

`restic` stores symlinks as symlinks by default and does not follow them during backup. This behavior helps avoid recursive traversal from link loops. If symlink following is enabled explicitly in a restic invocation, traversal/loop risk must be evaluated separately.
