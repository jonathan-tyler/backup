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
backup help
backup --help
```

Examples:

```sh
backup run daily
backup report weekly
backup report weekly new
backup report weekly excluded
backup restore /path/to/target
```

## Usage (wsl-sys-cli Extension)

When installed as an extension for `wsl-sys-cli`, run the same arguments through `sys backup`.

```sh
sys backup run daily
sys backup report weekly
sys backup report weekly new
sys backup report weekly excluded
sys backup restore /path/to/target
sys backup --help
```

## Development

```sh
go test ./...
go build ./...
```

Integration tests (tagged):

```sh
go test ./tests/unit/...
go test -tags=integration ./tests/integration/...
BACKUP_ITEST_PAUSE=1 go test -tags=integration -run TestIntegrationManifestAllCases -v ./tests/integration/...
```

Build output binaries in `out/` (ignored by git):

```sh
tests/manual/build_binaries.sh
ls -lh out/
```

Generated artifacts:

- `out/backup-linux-amd64`
- `out/backup-windows-amd64.exe`

For Windows manual testing, open this repo in Windows VS Code and run the Windows artifact directly.

Manual integration scripts in `tests/manual/` now build `out/` binaries first and pass `BACKUP_BINARY`
into integration tests. Set `SKIP_OUT_BUILD=1` to skip prebuild, or set `BACKUP_BINARY` explicitly to
override which binary is exercised.

## Configuration (Scaffold)

- Default config path in WSL/Linux: `~/.config/backup/config.yaml`
- Default config path in Windows: `%APPDATA%\\backup\\config.yaml`
- Optional override for both: `BACKUP_CONFIG=/custom/path/config.yaml`
- Starter template: [config.example.yaml](config.example.yaml)

Current status: config loading/validation is scaffolded for `run` planning only.
Backup execution is not implemented yet.
