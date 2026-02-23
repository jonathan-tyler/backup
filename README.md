# WSL Backup

Simple wrapper around `restic`. Extension for `wsl-sys-cli`.

## Focus

- WSL-first: launch from WSL and output to Windows filesystem
- Developed as an extension for wsl-sys-cli
- Denylist approach via exclude & include lists
- Daily, weekly, monthly commands with different retention policies
- Diff report for new items and another for excluded items
- Integration with keepass-xc

## Development Features

- Dev container for deterministic environment automation and isolation
- Test sandbox
- Unit & integration tests

### Optional Starship Host Config in Dev Container

If you personally use Starship and want to reuse your host `starship.toml`, uncomment the optional mount
line in [.devcontainer/devcontainer.json](.devcontainer/devcontainer.json).

## Installation

Build and install the executable as `backup`:

```sh
go build -o backup .
sudo install -m 0755 backup /usr/local/bin/backup
```

Verify:

```sh
backup --help
```

## Usage (Standalone Executable)

Run the CLI directly as `backup`.

```text
backup run <daily|weekly|monthly>
backup report <daily|weekly|monthly> [new|excluded]
backup restore <target>
backup test
backup help
backup --help
```

Platform behavior:

- This CLI is WSL-only and must be launched from a WSL window.
- Running from a Dev Container window or native Windows returns an error.
- In WSL, `backup run <cadence>` executes `wsl` and `windows` profiles in parallel.
- When WSL runs both platforms, the CLI warns if include paths appear to overlap across profiles
  (for example `/mnt/c/Users/...` and `C:\Users\...`).
- You can translate/compare path forms with `wslpath <path>` and `wslpath -w <path>`.
- Overlap verification runs in strict mode by default; runs fail when overlap is detected.

Examples:

```sh
backup run daily
backup report weekly
backup report weekly new
backup report weekly excluded
backup restore /path/to/target
backup test
```

## Usage (wsl-sys-cli Extension)

When installed as an extension for `wsl-sys-cli`, run the same arguments through `sys backup`.

```sh
sys backup run daily
sys backup report weekly
sys backup report weekly new
sys backup report weekly excluded
sys backup restore /path/to/target
sys backup test
sys backup --help
```

## Development

```sh
go test ./...
go build ./...
```

Integration tests (tagged):

- Full integration suite:

```sh
go test ./tests/unit/...
go test -tags=integration ./tests/integration/...
```

- Targeted restore-only integration check (faster while iterating on restore):

```sh
go test -tags=integration -run TestIntegrationRestoreLatest -v ./tests/integration/...
```

- Manifest pause mode for manual inspection:

```sh
BACKUP_ITEST_PAUSE=1 go test -tags=integration -run TestIntegrationManifestAllCases -v ./tests/integration/...
```

Build output binaries in `out/` (ignored by git):

```sh
tests/manual/build_binaries.sh
ls -lh out/
```

Generated artifacts:

- `out/backup-linux-amd64`
- `out/manual-itest-linux-amd64`
- `out/manual-itest-windows-amd64.exe`

Primary workflow is WSL-first with one installed `backup` entrypoint in WSL and `restic` installed in
both WSL and Windows. Do not run this CLI from native Windows.

Install/pin `restic` across WSL Fedora and Windows (Scoop) with repo canonical pin:

- Canonical repo pin: `scripts/restic-version.yaml`

```sh
scripts/install_restic_wsl_fedora.sh
```

This WSL-first script validates that the pinned version exists in both `dnf` and `scoop.exe`,
installs both Linux and Windows restic to that version, and scaffolds config/rules files when missing.

Start testing from WSL with:

```sh
go test ./tests/unit/...
go test -tags=integration ./tests/integration/...
tests/manual/run_manual_integration_tests.sh
backup test
```

For cross-platform validation, keep launching tests from WSL so one CLI orchestrates both platforms.

`backup test` runs Linux manual integration tests first, then Windows manual integration tests, and
pauses between phases.

Update to the newest cross-available version (dnf latest that matches scoop manifest):

```sh
scripts/update_restic_version.sh
```

This updates `scripts/restic-version.yaml`, and then runs
`scripts/install_restic_wsl_fedora.sh`.

Manual integration scripts in `tests/manual/` use prebuilt binaries from `out/` and pass `BACKUP_BINARY`
into integration tests. If `go` is available, they build fresh artifacts automatically before running.
If `go` is not available, they require existing `out/` artifacts and warn that prebuilt binaries may be
out of date; build first in the dev container with `tests/manual/build_binaries.sh` when needed.

Link behavior note: restic archives symlinks as symlinks by default and does not follow them during
backup. This project relies on that default behavior to avoid recursive traversal issues from symlink
loops. If follow-symlink behavior is enabled explicitly in a restic invocation, link traversal semantics
change and loop/cycle risk must be evaluated separately.

## Configuration (Scaffold)

- Default config path in WSL: `~/.config/backup/config.yaml`
- Optional override for both: `BACKUP_CONFIG=/custom/path/config.yaml`
- Starter template: [config.example.yaml](config.example.yaml)
- Restic pin: `scripts/restic-version.yaml`
- Rule files are auto-discovered next to config in `rules/` using this naming pattern:
 `<profile>.<include|exclude>.<daily|weekly|monthly>.txt`
- Rule files use one path per line (`#` comments allowed)
- Optional overrides are available with `include_files` and `exclude_files` in config when needed

Current status: config loading/validation is scaffolded for `run` planning only.
`run daily` executes planned restic invocations when config exists; report modes remain scaffolded.
