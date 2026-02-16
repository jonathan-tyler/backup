# Backup Run Guide

Current scope is `notes + repos` into `r-hot-notes-repos`.

## 1) One-time setup

1. Copy the environment template:

   ```bash
   cp config/backup.env.example config/backup.env
   ```

2. Edit `config/backup.env` for your local paths and KeePassXC entry name.
3. Ensure dependencies are installed:

   ```bash
   command -v restic
   command -v keepassxc-cli
   ```

4. Install the CLI with `pipx` from this repo:

   ```bash
   pipx install /home/him/my/sys/linux/backup
   ```

## 2) Daily operations

Run backup:

```bash
wsl-backup backup
```

Generate report:

```bash
wsl-backup report
```

## 3) Weekly operations

Apply retention:

```bash
wsl-backup forget-prune
```

Run restore smoke test:

```bash
wsl-backup restore-smoke
```

Use a custom env file by adding a second positional argument:

```bash
wsl-backup backup /path/to/backup.env
```

## 4) Optional cron examples (WSL)

```bash
# Daily backup at 21:00
0 21 * * * cd /home/him/my/sys/linux/backup && wsl-backup backup

# Daily report at 21:20
20 21 * * * cd /home/him/my/sys/linux/backup && wsl-backup report

# Weekly prune + restore test Sunday 10:00
0 10 * * 0 cd /home/him/my/sys/linux/backup && \
   wsl-backup forget-prune && wsl-backup restore-smoke
```

## 5) Output locations

- Backup repo: `RESTIC_REPOSITORY` in `config/backup.env`
- Reports and logs: `REPORT_DIR` in `config/backup.env`

## 6) Test commands

Fast unit tests:

```bash
pytest -q tests -k "not integration"
```

Integration tests (requires `restic` in container):

```bash
pytest -q tests -k integration
```

Full suite:

```bash
pytest -q
```
