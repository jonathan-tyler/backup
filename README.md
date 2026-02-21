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
